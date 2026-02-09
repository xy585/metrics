package main

import (
	"context"
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

func randomString(n int) string {
	b := make([]byte, n)
	rand.Read(b)
	return hex.EncodeToString(b)[:n]
}

func main() {
	kubeconfigPath := flag.String("kubeconfig", "./kubeconfig", "kubeconfig path")
	concurrency := flag.Int("c", 100, "(Concurrency)")
	total := flag.Int("n", 500000, "(Total)")
	flag.Parse()
	namespace := "default"

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

	fmt.Printf("start testing: total %d, concurrency %d\n", *total, *concurrency)

	jobs := make(chan int)
	var wg sync.WaitGroup

	for w := 0; w < *concurrency; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for id := range jobs {
				managedBy := fmt.Sprintf("t.io/%s-%s", time.Now().Format("150405"), randomString(40))
				if len(managedBy) > 63 {
					managedBy = managedBy[:63]
				}
				jobName := fmt.Sprintf("perf-%d-%s", id, randomString(8))

				job := &batchv1.Job{
					ObjectMeta: metav1.ObjectMeta{Name: jobName, Namespace: namespace},
					Spec: batchv1.JobSpec{
						ManagedBy: &managedBy,
						Template: corev1.PodTemplateSpec{
							Spec: corev1.PodSpec{
								RestartPolicy: corev1.RestartPolicyNever,
								Containers:    []corev1.Container{{Name: "w", Image: "busybox", Command: []string{"echo"}}},
							},
						},
					},
				}

				t1 := time.Now()
				_, err := clientset.BatchV1().Jobs(namespace).Create(context.TODO(), job, metav1.CreateOptions{})
				if err != nil {
					fmt.Printf("fail [%d]: %v\n", id, err)
					continue
				}

				policy := metav1.DeletePropagationBackground
				err = clientset.BatchV1().Jobs(namespace).Delete(context.TODO(), jobName, metav1.DeleteOptions{
					PropagationPolicy: &policy,
				})

				if id%10 == 0 {
					fmt.Printf("processed %d, current cost: %v\n", id, time.Since(t1))
				}
			}
		}()
	}

	go func() {
		for i := 0; i < *total; i++ {
			jobs <- i
		}
		close(jobs)
	}()

	wg.Wait()
	fmt.Printf("-----------------------------------\n")
	fmt.Printf("all done! cost: %v, avg: %.2f\n", time.Since(startTotal), float64(*total)/time.Since(startTotal).Seconds())
}
