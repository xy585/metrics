package main

import (
	"context"
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"strings"
	"sync"
	"time"

	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

func randomString(n int) string {
	b := make([]byte, n)
	rand.Read(b)
	return hex.EncodeToString(b)[:n]
}

func generateLargeName(id int) string {

	const targetSize = 500 * 1024 // 500KB

	prefix := fmt.Sprintf("clusterrole-%d-", id)

	remaining := targetSize - len(prefix)

	randomPart := strings.Repeat(randomString(1000), remaining/1000)
	randomPart += randomString(remaining % 1000)

	return prefix + randomPart
}

func main() {
	kubeconfigPath := flag.String("kubeconfig", "./kubeconfig", "kubeconfig path")
	concurrency := flag.Int("c", 100, "(Concurrency)")
	total := flag.Int("n", 20000, "(Total)")
	flag.Parse()

	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfigPath)
	if err != nil {
		log.Fatal(err)
	}

	config.QPS = 500
	config.Burst = 500

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatal(err)
	}

	startTotal := time.Now()

	fmt.Printf("start testing: total requests %d, concurrency %d\n", *total, *concurrency)
	fmt.Printf("each ClusterRole name size: about 500KB\n\n")

	jobs := make(chan int)
	var wg sync.WaitGroup
	var successCount, failCount int
	var mu sync.Mutex

	for w := 0; w < *concurrency; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for id := range jobs {
				clusterRoleName := generateLargeName(id)
				clusterRole := &rbacv1.ClusterRole{
					ObjectMeta: metav1.ObjectMeta{
						Name: clusterRoleName,
					},
					Rules: []rbacv1.PolicyRule{
						{
							APIGroups: []string{""},
							Resources: []string{"pods"},
							Verbs:     []string{"get", "list", "watch"},
						},
					},
				}

				t1 := time.Now()
				_, err := clientset.RbacV1().ClusterRoles().Create(context.TODO(), clusterRole, metav1.CreateOptions{})
				duration := time.Since(t1)

				mu.Lock()
				if err != nil {
					failCount++
					fmt.Printf("X [%d] fail (duration: %v): %v\n", id, duration, err)
				} else {
					successCount++
					if id%10 == 0 || id < 10 {
						fmt.Printf("√ [%d] success (duration: %v, name length: %d bytes)\n",
							id, duration, len(clusterRoleName))
					}

				}
				mu.Unlock()
			}
		}(w)
	}

	go func() {
		for i := 0; i < *total; i++ {
			jobs <- i
		}
		close(jobs)
	}()

	wg.Wait()
	totalDuration := time.Since(startTotal)

	fmt.Printf("\n" + strings.Repeat("=", 60) + "\n")
	fmt.Printf("all done!\n")
	fmt.Printf("cost: %v\n", totalDuration)
	fmt.Printf("success: %d\n", successCount)
	fmt.Printf("fail: %d\n", failCount)
	fmt.Printf("total: %d\n", *total)
	if successCount > 0 {
		fmt.Printf("avg: %v\n", totalDuration/time.Duration(successCount))
		fmt.Printf("QPS: %.2f\n", float64(successCount)/totalDuration.Seconds())
	}
	fmt.Printf(strings.Repeat("=", 60) + "\n")
}
