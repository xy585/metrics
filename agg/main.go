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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/clientcmd"
)

func randomString(n int) string {
	b := make([]byte, n/2+1)
	rand.Read(b)
	return hex.EncodeToString(b)[:n]
}

func generateLargeName(sizeKB int, prefix string) string {
	targetSize := sizeKB * 1024

	var builder strings.Builder
	builder.WriteString(prefix)

	remaining := targetSize - len(prefix)
	if remaining > 0 {
		randomPart := randomString(remaining)
		builder.WriteString(randomPart)
	}

	return builder.String()
}

func main() {
	kubeconfigPath := flag.String("kubeconfig", "./kubeconfig", "kubeconfig file path")
	concurrency := flag.Int("c", 100, "concurrency")
	total := flag.Int("n", 1000, "total number of requests")
	nameSizeKB := flag.Int("size", 1000, "name size (KB)")
	flag.Parse()

	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfigPath)
	if err != nil {
		log.Fatal(err)
	}

	config.QPS = 500
	config.Burst = 500

	dynamicClient, err := dynamic.NewForConfig(config)
	restClient := dynamicClient.Resource(schema.GroupVersionResource{})
	if err != nil {
		log.Fatal(err)
	}

	startTotal := time.Now()

	jobs := make(chan int)
	var wg sync.WaitGroup

	successCount := 0
	failCount := 0
	var mu sync.Mutex

	for w := 0; w < *concurrency; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for id := range jobs {
				prefix := fmt.Sprintf("testagg-%d-%s-", id, time.Now().Format("20060102-150405"))

				rcName := generateLargeName(*nameSizeKB, prefix)
				//fmt.Println(rcName)
				t1 := time.Now()
				restClient = dynamicClient.Resource(schema.GroupVersionResource{
					Group:    "example.com",
					Version:  "v1alpha1",
					Resource: rcName,
				})
				_, err := restClient.Namespace("default").Get(context.TODO(), "tmp", metav1.GetOptions{})
				if err != nil {
					//anic(err)
				}

				mu.Lock()
				if err != nil {
					failCount++
					// fmt.Printf("[fail %d] name length: %d bytes, error: %v, duration: %v\n",
					// 	id, len(rcName), err, time.Since(t1))
				} else {
					successCount++
					if id%10 == 0 {
						fmt.Printf("[success %d] name length: %d bytes, duration: %v\n",
							id, len(rcName), time.Since(t1))
					}
				}
				mu.Unlock()
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

	fmt.Printf("===================================\n")
	fmt.Printf("Test completed!\n")
	fmt.Printf("Total duration: %v\n", time.Since(startTotal))
	fmt.Printf("Success: %d, Fail: %d\n", successCount, failCount)
	fmt.Printf("Average QPS: %.2f\n", float64(*total)/time.Since(startTotal).Seconds())
}
