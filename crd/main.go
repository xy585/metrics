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

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiextensionsclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
)

func randomString(n int) string {
	b := make([]byte, n)
	rand.Read(b)
	return hex.EncodeToString(b)[:n]
}

func main() {
	kubeconfigPath := flag.String("kubeconfig", "./kubeconfig", "kubeconfig path")
	concurrency := flag.Int("c", 50, "(Concurrency Concurrency)")
	total := flag.Int("n", 100000, "(Total)")
	flag.Parse()

	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfigPath)
	if err != nil {
		log.Fatal(err)
	}

	config.QPS = 500
	config.Burst = 500

	clientset, err := apiextensionsclientset.NewForConfig(config)
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
				str61 := "a" + randomString(60)
				group := str61 + ".a"
				plural := fmt.Sprintf("%s", str61+"as")
				singular := str61 + "a"
				kind := str61
				crdName := fmt.Sprintf("%s.%s", plural, group)

				crd := &apiextensionsv1.CustomResourceDefinition{
					ObjectMeta: metav1.ObjectMeta{
						Name: crdName,
					},
					Spec: apiextensionsv1.CustomResourceDefinitionSpec{
						Group: group,
						Names: apiextensionsv1.CustomResourceDefinitionNames{
							Plural:   plural,
							Singular: singular,
							Kind:     kind,
							ListKind: kind + "s",
						},
						Scope: apiextensionsv1.ClusterScoped,
						Versions: []apiextensionsv1.CustomResourceDefinitionVersion{
							{
								Name:    "v3333333333333333333333333333333333333333333333333333333333333",
								Served:  true,
								Storage: true,
								Schema: &apiextensionsv1.CustomResourceValidation{
									OpenAPIV3Schema: &apiextensionsv1.JSONSchemaProps{
										Type: "object",
										Properties: map[string]apiextensionsv1.JSONSchemaProps{
											"spec": {
												Type: "object",
												Properties: map[string]apiextensionsv1.JSONSchemaProps{
													"field": {Type: "string"},
												},
											},
										},
									},
								},
							},
						},
					},
				}

				t1 := time.Now()
				_, err := clientset.ApiextensionsV1().CustomResourceDefinitions().Create(context.TODO(), crd, metav1.CreateOptions{})
				if err != nil {
					fmt.Printf("fail [%d]: %v\n", id, err)
					continue
				}

				//policy := metav1.DeletePropagationBackground
				err = clientset.ApiextensionsV1().CustomResourceDefinitions().Delete(context.TODO(), crdName, metav1.DeleteOptions{
					//PropagationPolicy: &policy,
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
