package visualizer

import (
	"context"
	"fmt"
	"os"

	"github.com/goccy/go-graphviz"
	"github.com/goccy/go-graphviz/cgraph"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type DependencyVisualizer struct {
	clientset *kubernetes.Clientset
	namespace string
}

func New(clientset *kubernetes.Clientset, namespace string) *DependencyVisualizer {
	return &DependencyVisualizer{
		clientset: clientset,
		namespace: namespace,
	}
}

func (v *DependencyVisualizer) CreateGraph(outputPath string) error {
	ctx := context.Background()
	g, err := graphviz.New(ctx)
	if err != nil {
		return fmt.Errorf("error creating graphviz instance: %v", err)
	}
	defer func() {
		if err := g.Close(); err != nil {
			fmt.Printf("error closing graphviz: %v\n", err)
		}
	}()

	graph, err := g.Graph()
	if err != nil {
		return fmt.Errorf("error creating graph: %v", err)
	}
	defer func() {
		if err := graph.Close(); err != nil {
			fmt.Printf("error closing graph: %v\n", err)
		}
	}()

	// Create nodes map to store references
	nodes := make(map[string]*cgraph.Node)

	// Get deployments
	deployments, err := v.clientset.AppsV1().Deployments(v.namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("error listing deployments: %v", err)
	}

	// Add deployment nodes
	for _, deployment := range deployments.Items {
		node, err := graph.CreateNodeByName(deployment.Name)
		if err != nil {
			return fmt.Errorf("error creating deployment node: %v", err)
		}
		if err := node.Set("style", "filled"); err != nil {
			return fmt.Errorf("error setting node style: %v", err)
		}
		if err := node.Set("fillcolor", "lightblue"); err != nil {
			return fmt.Errorf("error setting node color: %v", err)
		}
		nodes[fmt.Sprintf("deployment/%s", deployment.Name)] = node
	}

	// Get services
	services, err := v.clientset.CoreV1().Services(v.namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("error listing services: %v", err)
	}

	// Add service nodes
	for _, service := range services.Items {
		node, err := graph.CreateNodeByName(service.Name)
		if err != nil {
			return fmt.Errorf("error creating service node: %v", err)
		}
		if err := node.Set("style", "filled"); err != nil {
			return fmt.Errorf("error setting node style: %v", err)
		}
		if err := node.Set("fillcolor", "lightgreen"); err != nil {
			return fmt.Errorf("error setting node color: %v", err)
		}
		nodes[fmt.Sprintf("service/%s", service.Name)] = node

		// Connect services to deployments
		if service.Spec.Selector != nil {
			for _, deployment := range deployments.Items {
				if labelsMatch(deployment.Spec.Template.Labels, service.Spec.Selector) {
					deploymentNode := nodes[fmt.Sprintf("deployment/%s", deployment.Name)]
					if deploymentNode != nil {
						edge, err := graph.CreateEdgeByName("", deploymentNode, node)
						if err != nil {
							return fmt.Errorf("error creating edge: %v", err)
						}
						if err := edge.Set("label", "selects"); err != nil {
							return fmt.Errorf("error setting edge label: %v", err)
						}
					}
				}
			}
		}
	}

	// Get configmaps
	configmaps, err := v.clientset.CoreV1().ConfigMaps(v.namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("error listing configmaps: %v", err)
	}

	// Add configmap nodes
	for _, configmap := range configmaps.Items {
		node, err := graph.CreateNodeByName(configmap.Name)
		if err != nil {
			return fmt.Errorf("error creating configmap node: %v", err)
		}
		if err := node.Set("style", "filled"); err != nil {
			return fmt.Errorf("error setting node style: %v", err)
		}
		if err := node.Set("fillcolor", "yellow"); err != nil {
			return fmt.Errorf("error setting node color: %v", err)
		}
		nodes[fmt.Sprintf("configmap/%s", configmap.Name)] = node
	}

	// Get secrets
	secrets, err := v.clientset.CoreV1().Secrets(v.namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("error listing secrets: %v", err)
	}

	// Add secret nodes
	for _, secret := range secrets.Items {
		node, err := graph.CreateNodeByName(secret.Name)
		if err != nil {
			return fmt.Errorf("error creating secret node: %v", err)
		}
		if err := node.Set("style", "filled"); err != nil {
			return fmt.Errorf("error setting node style: %v", err)
		}
		if err := node.Set("fillcolor", "pink"); err != nil {
			return fmt.Errorf("error setting node color: %v", err)
		}
		nodes[fmt.Sprintf("secret/%s", secret.Name)] = node
	}

	// Connect deployments to configmaps and secrets
	for _, deployment := range deployments.Items {
		deploymentNode := nodes[fmt.Sprintf("deployment/%s", deployment.Name)]
		if deploymentNode == nil {
			continue
		}

		// Check volumes
		for _, volume := range deployment.Spec.Template.Spec.Volumes {
			if volume.ConfigMap != nil {
				configmapNode := nodes[fmt.Sprintf("configmap/%s", volume.ConfigMap.Name)]
				if configmapNode != nil {
					edge, err := graph.CreateEdgeByName("", configmapNode, deploymentNode)
					if err != nil {
						return fmt.Errorf("error creating edge: %v", err)
					}
					if err := edge.Set("label", "mounts"); err != nil {
						return fmt.Errorf("error setting edge label: %v", err)
					}
				}
			}
			if volume.Secret != nil {
				secretNode := nodes[fmt.Sprintf("secret/%s", volume.Secret.SecretName)]
				if secretNode != nil {
					edge, err := graph.CreateEdgeByName("", secretNode, deploymentNode)
					if err != nil {
						return fmt.Errorf("error creating edge: %v", err)
					}
					if err := edge.Set("label", "mounts"); err != nil {
						return fmt.Errorf("error setting edge label: %v", err)
					}
				}
			}
		}

		// Check environment variables
		for _, container := range deployment.Spec.Template.Spec.Containers {
			for _, env := range container.EnvFrom {
				if env.ConfigMapRef != nil {
					configmapNode := nodes[fmt.Sprintf("configmap/%s", env.ConfigMapRef.Name)]
					if configmapNode != nil {
						edge, err := graph.CreateEdgeByName("", configmapNode, deploymentNode)
						if err != nil {
							return fmt.Errorf("error creating edge: %v", err)
						}
						if err := edge.Set("label", "env"); err != nil {
							return fmt.Errorf("error setting edge label: %v", err)
						}
					}
				}
				if env.SecretRef != nil {
					secretNode := nodes[fmt.Sprintf("secret/%s", env.SecretRef.Name)]
					if secretNode != nil {
						edge, err := graph.CreateEdgeByName("", secretNode, deploymentNode)
						if err != nil {
							return fmt.Errorf("error creating edge: %v", err)
						}
						if err := edge.Set("label", "env"); err != nil {
							return fmt.Errorf("error setting edge label: %v", err)
						}
					}
				}
			}
		}
	}

	// Create output file
	out, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("error creating output file: %v", err)
	}
	defer out.Close()

	// Render graph
	if err := g.Render(ctx, graph, graphviz.PNG, out); err != nil {
		return fmt.Errorf("error rendering graph: %v", err)
	}

	return nil
}

func labelsMatch(labels, selector map[string]string) bool {
	for key, value := range selector {
		if labels[key] != value {
			return false
		}
	}
	return true
}
