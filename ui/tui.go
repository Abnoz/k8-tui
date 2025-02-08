package ui

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/k8s-admin-cli/visualizer"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"k8s.io/metrics/pkg/client/clientset/versioned"
)

var (
	kubeconfig string
	program    *tea.Program
	namespace  string
)

func init() {
	if home := homedir.HomeDir(); home != "" {
		kubeconfig = filepath.Join(home, ".kube", "config")
	}
}

type item struct {
	title, description string
}

func (i item) Title() string       { return i.title }
func (i item) Description() string { return i.description }
func (i item) FilterValue() string { return i.title }

type model struct {
	list           list.Model
	submenuItems   list.Model
	roleItems      list.Model
	podItems       list.Model
	spinner        spinner.Model
	viewport       viewport.Model
	textInput      textinput.Model
	width          int
	height         int
	loading        bool
	err            error
	result         string
	selectedItem   string
	inSubmenu      bool
	inRoleMenu     bool
	inPodMenu      bool
	inputting      bool
	inputAction    string
	inputStep      string
	inputData      map[string]string
	rules          []rbacv1.PolicyRule
	selectedPod    string
	resourceItems  list.Model
	inResourceMenu bool
}

type itemDelegate struct{}

func (d itemDelegate) Height() int                             { return 1 }
func (d itemDelegate) Spacing() int                            { return 0 }
func (d itemDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d itemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(item)
	if !ok {
		return
	}

	str := i.title
	if i.description != "" {
		str = fmt.Sprintf("%s - %s", i.title, i.description)
	}

	fn := lipgloss.NewStyle().Render
	if index == m.Index() {
		fn = func(strs ...string) string {
			// Implement logic to handle multiple strings
			return lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Render("> " + strings.Join(strs, " "))
		}
	}

	fmt.Fprint(w, fn(str))
}

func createMainList() list.Model {
	items := []list.Item{
		item{title: "Pods", description: "Manage Kubernetes pods"},
		item{title: "Service Accounts", description: "Manage Kubernetes service accounts"},
		item{title: "Roles", description: "Manage RBAC roles"},
		item{title: "Role Bindings", description: "Manage RBAC role bindings"},
		item{title: "Health Check", description: "View cluster health status"},
		item{title: "Resource Analyzer", description: "Analyze and optimize resource usage across pods"},
		item{title: "Visualize Dependencies", description: "Generate a visual graph of resource dependencies"},
		item{title: "Quit", description: "Exit the application"},
	}

	l := list.New(items, list.NewDefaultDelegate(), 0, 0)
	l.Title = "Kubernetes Admin Console"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.Styles.Title = lipgloss.NewStyle().
		Foreground(lipgloss.Color("205")).
		Bold(true).
		MarginLeft(2)
	l.Styles.PaginationStyle = lipgloss.NewStyle().Padding(0, 1)

	return l
}

func createResourceSubmenu() list.Model {
	items := []list.Item{
		item{title: "Pods", description: "Manage Kubernetes pods"},
		item{title: "Service Accounts", description: "Manage Kubernetes service accounts"},
		item{title: "Roles", description: "Manage RBAC roles"},
		item{title: "Role Bindings", description: "Manage RBAC role bindings"},
		item{title: "Back", description: "Return to main menu"},
	}

	l := list.New(items, itemDelegate{}, 0, 0)
	l.Title = "Resource Management"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.Styles.Title = lipgloss.NewStyle().
		Foreground(lipgloss.Color("205")).
		Bold(true).
		MarginLeft(2)
	l.Styles.PaginationStyle = lipgloss.NewStyle().Padding(0, 1)

	return l
}

func createServiceAccountSubmenu() list.Model {
	saSubItems := []list.Item{
		item{title: "List Service Accounts", description: "View all service accounts"},
		item{title: "Create Service Account", description: "Create a new service account"},
		item{title: "Delete Service Account", description: "Delete an existing service account"},
		item{title: "Back", description: "Return to main menu"},
	}

	saList := list.New(saSubItems, list.NewDefaultDelegate(), 0, 0)
	saList.Title = "Service Account Operations"
	saList.SetShowStatusBar(false)
	saList.SetFilteringEnabled(false)
	saList.Styles.Title = lipgloss.NewStyle().
		Foreground(lipgloss.Color("205")).
		Bold(true).
		MarginLeft(2)
	saList.Styles.PaginationStyle = lipgloss.NewStyle().Padding(0, 1)

	return saList
}

func createRoleSubmenu() list.Model {
	items := []list.Item{
		item{title: "List Roles", description: "View all roles"},
		item{title: "Create Role", description: "Create a new role"},
		item{title: "Delete Role", description: "Delete an existing role"},
		item{title: "Back", description: "Return to main menu"},
	}

	l := list.New(items, list.NewDefaultDelegate(), 0, 0)
	l.Title = "Role Management"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.Styles.Title = lipgloss.NewStyle().
		Foreground(lipgloss.Color("205")).
		Bold(true).
		MarginLeft(2)
	l.Styles.PaginationStyle = lipgloss.NewStyle().Padding(0, 1)

	return l
}

func createPodSubmenu() list.Model {
	podSubItems := []list.Item{
		item{title: "List Pods", description: "View all pods"},
		item{title: "Create Pod", description: "Create a new pod"},
		item{title: "Delete Pod", description: "Delete an existing pod"},
		item{title: "Pod Details", description: "View detailed pod information"},
		item{title: "Pod Logs", description: "View pod logs"},
		item{title: "Back", description: "Return to main menu"},
	}

	podList := list.New(podSubItems, list.NewDefaultDelegate(), 0, 0)
	podList.Title = "Pod Operations"
	podList.SetShowStatusBar(false)
	podList.SetFilteringEnabled(false)
	podList.Styles.Title = lipgloss.NewStyle().
		Foreground(lipgloss.Color("205")).
		Bold(true).
		MarginLeft(2)
	podList.Styles.PaginationStyle = lipgloss.NewStyle().Padding(0, 1)

	return podList
}

func initialModel() *model {
	mainList := createMainList()
	resourceList := createResourceSubmenu()
	saList := createServiceAccountSubmenu()
	roleList := createRoleSubmenu()
	podList := createPodSubmenu()

	vp := viewport.New(100, 30)
	vp.Style = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Padding(1, 2)

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	ti := textinput.New()
	ti.Placeholder = "Enter value..."
	ti.CharLimit = 156
	ti.Width = 40
	ti.Focus()

	m := &model{
		list:          mainList,
		resourceItems: resourceList,
		submenuItems:  saList,
		roleItems:     roleList,
		podItems:      podList,
		textInput:     ti,
		spinner:       s,
		viewport:      vp,
		inputData:     make(map[string]string),
		width:         100,
		height:        30,
	}

	m.list.SetWidth(100 - 4)
	m.list.SetHeight(30 - 4)
	m.submenuItems.SetWidth(100 - 4)
	m.submenuItems.SetHeight(30 - 4)
	m.roleItems.SetWidth(100 - 4)
	m.roleItems.SetHeight(30 - 4)
	m.podItems.SetWidth(100 - 4)
	m.podItems.SetHeight(30 - 4)
	m.resourceItems.SetWidth(100 - 4)
	m.resourceItems.SetHeight(30 - 4)

	return m
}

func New() *tea.Program {
	p := tea.NewProgram(
		initialModel(),
		tea.WithAltScreen(),
	)
	program = p
	return p
}

func (m *model) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		tea.EnterAltScreen,
		textinput.Blink,
	)
}

func (m *model) clearResults() {
	m.result = ""
	m.viewport.SetContent("")
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.viewport.Width = msg.Width - 4
		m.viewport.Height = msg.Height - 4
		return m, nil

	case tea.KeyMsg:
		if m.loading {
			return m, nil
		}

		if m.inputting {
			switch msg.Type {
			case tea.KeyEsc:
				m.inputting = false
				m.textInput.Blur()
				m.clearResults()
				return m, nil
			case tea.KeyEnter:
				value := m.textInput.Value()
				m.handleInput(value)
				m.textInput.SetValue("")
				return m, nil
			}
			m.textInput, cmd = m.textInput.Update(msg)
			return m, cmd
		}

		switch keypress := msg.String(); keypress {
		case "q":
			if m.result != "" {
				m.clearResults()
				return m, nil
			}
			if m.inPodMenu || m.inSubmenu || m.inRoleMenu {
				m.inPodMenu = false
				m.inSubmenu = false
				m.inRoleMenu = false
				m.clearResults()
				return m, nil
			}
		case "ctrl+c":
			return m, tea.Quit
		case "enter":
			return m.handleEnterKey()
		}

		if m.result != "" {
			m.viewport, cmd = m.viewport.Update(msg)
			return m, cmd
		}

		if m.inPodMenu {
			m.podItems, cmd = m.podItems.Update(msg)
		} else if m.inSubmenu {
			m.submenuItems, cmd = m.submenuItems.Update(msg)
		} else if m.inRoleMenu {
			m.roleItems, cmd = m.roleItems.Update(msg)
		} else {
			m.list, cmd = m.list.Update(msg)
		}
		return m, cmd
	}

	return m, nil
}

func (m *model) View() string {
	var content string

	if m.err != nil {
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Render(fmt.Sprintf("Error: %v\nPress any key to continue", m.err))
	}

	if m.loading {
		content = fmt.Sprintf("\n\n   %s Loading...\n\n", m.spinner.View())
	} else if m.err != nil {
		content = fmt.Sprintf("\n\n  Error: %v\nPress 'q' to return to menu\n", m.err)
	} else if m.inputting {
		content = fmt.Sprintf("\n\n  %s\n\n  %s", m.result, m.textInput.View())
		return lipgloss.NewStyle().
			MaxWidth(m.width - 4).
			MaxHeight(m.height - 4).
			Render(content)
	} else if m.result != "" {
		content = m.viewport.View()
	} else if m.inResourceMenu {
		content = "\n" + m.resourceItems.View()
	} else if m.inPodMenu {
		content = "\n" + m.podItems.View()
	} else if m.inSubmenu {
		content = "\n" + m.submenuItems.View()
	} else if m.inRoleMenu {
		content = "\n" + m.roleItems.View()
	} else {
		content = "\n" + m.list.View()
	}

	helpText := "\nNavigate: ↑/k ↓/j • Scroll: PgUp/PgDown • Top/Bottom: Home/End • Back: q"

	return lipgloss.NewStyle().
		MaxWidth(m.width - 4).
		MaxHeight(m.height - 4).
		Render(content + helpText)
}

func (m *model) handleInput(value string) {
	switch m.inputAction {
	case "create-pod":
		switch m.inputStep {
		case "name":
			m.inputData["name"] = value
			m.inputStep = "image"
			m.textInput.SetValue("")
			m.textInput.Placeholder = "Enter container image..."
			m.textInput.Focus()
			m.result = "Enter container image (press Enter to confirm, Esc to cancel):"
		case "image":
			m.inputData["image"] = value
			m.inputStep = "namespace"
			m.textInput.SetValue("")
			m.textInput.Placeholder = "Enter namespace (default: default)..."
			m.textInput.Focus()
			m.result = "Enter namespace (press Enter to confirm, Esc to cancel):"
		case "namespace":
			if value == "" {
				value = "default"
			}
			m.inputData["namespace"] = value
			m.inputStep = "confirm"
			m.textInput.SetValue("")
			m.textInput.Placeholder = "Type 'y' to confirm or 'n' to cancel..."
			m.textInput.Focus()
			m.result = fmt.Sprintf("Create pod with the following details?\nName: %s\nImage: %s\nNamespace: %s\n(y/n)",
				m.inputData["name"], m.inputData["image"], m.inputData["namespace"])
		case "confirm":
			if strings.ToLower(value) == "y" {
				go m.createPod()
			} else {
				m.result = "Pod creation cancelled."
				m.inputting = false
			}
		}
	case "delete-pod":
		m.result = fmt.Sprintf("Deleting pod %s", value)
		m.inputting = false
	case "pod-details":
		m.result = fmt.Sprintf("Fetching details for pod %s", value)
		m.inputting = false
	case "pod-logs":
		m.result = fmt.Sprintf("Fetching logs for pod %s", value)
		m.inputting = false
	case "create-sa":
		switch m.inputStep {
		case "name":
			m.inputData["name"] = value
			m.inputStep = "namespace"
			m.textInput.SetValue("")
			m.textInput.Placeholder = "Enter namespace (default: default)..."
			m.textInput.Focus()
			m.result = "Enter namespace for the service account (press Enter to confirm, Esc to cancel):"
		case "namespace":
			if value == "" {
				value = "default"
			}
			m.inputData["namespace"] = value
			m.inputStep = "confirm"
			m.textInput.SetValue("")
			m.textInput.Placeholder = "Type 'y' to confirm or 'n' to cancel..."
			m.textInput.Focus()
			m.result = fmt.Sprintf("Create service account with the following details?\nName: %s\nNamespace: %s\n(y/n)",
				m.inputData["name"], m.inputData["namespace"])
		case "confirm":
			if strings.ToLower(value) == "y" {
				go m.createServiceAccount()
			} else {
				m.result = "Service account creation cancelled."
				m.inputting = false
			}
		}
	case "delete-sa":
		m.result = fmt.Sprintf("Deleting service account %s", value)
		m.inputting = false
	case "create-role":
		m.result = fmt.Sprintf("Creating role %s", value)
		m.inputting = false
	case "delete-role":
		m.result = fmt.Sprintf("Deleting role %s", value)
		m.inputting = false
	}
	m.viewport.SetContent(m.result)
}

func (m *model) handleEnterKey() (tea.Model, tea.Cmd) {
	if m.inPodMenu {
		i, ok := m.podItems.SelectedItem().(item)
		if ok {
			switch i.title {
			case "List Pods":
				m.loading = true
				m.clearResults()
				go func() {
					clientset, err := getClientset()
					if err != nil {
						m.loading = false
						m.err = err
						program.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{0}})
						return
					}

					pods, err := clientset.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{})
					m.loading = false
					if err != nil {
						m.err = err
					} else {
						m.result = formatPodList(pods)
						m.viewport.SetContent(m.result)
					}
					program.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{0}})
				}()
				return m, m.spinner.Tick

			case "Create Pod":
				m.inputting = true
				m.inputAction = "create-pod"
				m.inputStep = "name"
				m.textInput.Reset()
				m.textInput.Placeholder = "Enter pod name..."
				m.textInput.Focus()
				m.result = "Enter name for the pod (press Enter to confirm, Esc to cancel):"
				m.viewport.SetContent(m.result)
				return m, nil

			case "Delete Pod":
				m.inputting = true
				m.inputAction = "delete-pod"
				m.inputStep = "name"
				m.textInput.Reset()
				m.textInput.Placeholder = "Enter pod name..."
				m.textInput.Focus()
				m.result = "Enter name of the pod to delete (press Enter to confirm, Esc to cancel):"
				m.viewport.SetContent(m.result)
				return m, nil

			case "Pod Details":
				m.inputting = true
				m.inputAction = "pod-details"
				m.inputStep = "name"
				m.textInput.Reset()
				m.textInput.Placeholder = "Enter pod name..."
				m.textInput.Focus()
				m.result = "Enter name of the pod to view details (press Enter to confirm, Esc to cancel):"
				m.viewport.SetContent(m.result)
				return m, nil

			case "Pod Logs":
				m.inputting = true
				m.inputAction = "pod-logs"
				m.inputStep = "name"
				m.textInput.Reset()
				m.textInput.Placeholder = "Enter pod name..."
				m.textInput.Focus()
				m.result = "Enter name of the pod to view logs (press Enter to confirm, Esc to cancel):"
				m.viewport.SetContent(m.result)
				return m, nil

			case "Back":
				m.inPodMenu = false
				m.clearResults()
				return m, nil
			}
		}
	} else if m.inSubmenu {
		i, ok := m.submenuItems.SelectedItem().(item)
		if ok {
			switch i.title {
			case "List Service Accounts":
				m.loading = true
				m.clearResults()
				go func() {
					clientset, err := getClientset()
					if err != nil {
						m.loading = false
						m.err = err
						program.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{0}})
						return
					}

					sas, err := clientset.CoreV1().ServiceAccounts(namespace).List(context.TODO(), metav1.ListOptions{})
					m.loading = false
					if err != nil {
						m.err = err
					} else {
						m.result = formatServiceAccountList(sas)
						m.viewport.SetContent(m.result)
					}
					program.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{0}})
				}()
				return m, m.spinner.Tick
			case "Create Service Account":
				m.inputting = true
				m.inputAction = "create-sa"
				m.inputStep = "name"
				m.textInput.Reset()
				m.textInput.Placeholder = "Enter service account name..."
				m.textInput.Focus()
				m.result = "Enter name for the service account (press Enter to confirm, Esc to cancel):"
				m.viewport.SetContent(m.result)
				return m, nil
			case "Delete Service Account":
				m.inputting = true
				m.inputAction = "delete-sa"
				m.inputStep = "name"
				m.textInput.Reset()
				m.textInput.Placeholder = "Enter service account name..."
				m.textInput.Focus()
				m.result = "Enter name of the service account to delete (press Enter to confirm, Esc to cancel):"
				m.viewport.SetContent(m.result)
				return m, nil
			case "Back":
				m.inSubmenu = false
				m.clearResults()
				return m, nil
			}
		}
	} else if m.inRoleMenu {
		i, ok := m.roleItems.SelectedItem().(item)
		if ok {
			switch i.title {
			case "List Roles":
				m.loading = true
				m.clearResults()
				go func() {
					clientset, err := getClientset()
					if err != nil {
						m.loading = false
						m.err = err
						program.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{0}})
						return
					}

					roles, err := clientset.RbacV1().Roles(namespace).List(context.TODO(), metav1.ListOptions{})
					m.loading = false
					if err != nil {
						m.err = err
					} else {
						m.result = formatRolesList(roles)
						m.viewport.SetContent(m.result)
					}
					program.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{0}})
				}()
				return m, m.spinner.Tick
			case "Create Role":
				m.inputting = true
				m.inputAction = "create-role"
				m.inputStep = "name"
				m.textInput.Reset()
				m.textInput.Placeholder = "Enter role name..."
				m.textInput.Focus()
				m.result = "Enter name for the role (press Enter to confirm, Esc to cancel):"
				m.viewport.SetContent(m.result)
				return m, nil
			case "Delete Role":
				m.inputting = true
				m.inputAction = "delete-role"
				m.inputStep = "name"
				m.textInput.Reset()
				m.textInput.Placeholder = "Enter role name..."
				m.textInput.Focus()
				m.result = "Enter name of the role to delete (press Enter to confirm, Esc to cancel):"
				m.viewport.SetContent(m.result)
				return m, nil
			case "Back":
				m.inRoleMenu = false
				m.clearResults()
				return m, nil
			}
		}
	} else {
		i, ok := m.list.SelectedItem().(item)
		if ok {
			switch i.title {
			case "Pods":
				m.inPodMenu = true
				m.inSubmenu = false
				m.inRoleMenu = false
				m.clearResults()
				return m, nil
			case "Service Accounts":
				m.inSubmenu = true
				m.inRoleMenu = false
				m.inPodMenu = false
				m.clearResults()
				return m, nil
			case "Roles":
				m.inRoleMenu = true
				m.inSubmenu = false
				m.inPodMenu = false
				m.clearResults()
				return m, nil
			case "Health Check":
				m.loading = true
				go func() {
					result, err := checkClusterHealth()
					m.loading = false
					if err != nil {
						m.err = err
					} else {
						m.result = result
						m.viewport.SetContent(m.result)
					}
					program.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{0}})
				}()
				return m, m.spinner.Tick
			case "Resource Analyzer":
				m.loading = true
				go func() {
					result, err := analyzeResourcesForTUI()
					m.loading = false
					if err != nil {
						m.err = err
					} else {
						m.result = result
						m.viewport.SetContent(m.result)
					}
					program.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{0}})
				}()
				return m, m.spinner.Tick
			case "Visualize Dependencies":
				m.loading = true
				go func() {
					result, err := generateDependencyGraph()
					m.loading = false
					if err != nil {
						m.err = err
					} else {
						m.result = result
						m.viewport.SetContent(m.result)
					}
					program.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{0}})
				}()
				return m, m.spinner.Tick
			case "Quit":
				return m, tea.Quit
			}
		}
	}
	return m, nil
}

func (m *model) createPod() {
	clientset, err := getClientset()
	if err != nil {
		m.result = fmt.Sprintf("Error getting clientset: %v", err)
		m.inputting = false
		return
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      m.inputData["name"],
			Namespace: m.inputData["namespace"],
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  m.inputData["name"],
					Image: m.inputData["image"],
				},
			},
		},
	}

	_, err = clientset.CoreV1().Pods(m.inputData["namespace"]).Create(context.TODO(), pod, metav1.CreateOptions{})
	if err != nil {
		m.result = fmt.Sprintf("Error creating pod: %v", err)
	} else {
		m.result = fmt.Sprintf("Pod %s created successfully in namespace %s", m.inputData["name"], m.inputData["namespace"])
	}
	m.inputting = false
	program.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{0}})
}

func (m *model) createServiceAccount() {
	err := createServiceAccount(m.inputData["name"], m.inputData["namespace"])
	if err != nil {
		m.result = fmt.Sprintf("Error creating service account: %v", err)
	} else {
		m.result = fmt.Sprintf("Service account %s created successfully in namespace %s", m.inputData["name"], m.inputData["namespace"])
	}
	m.inputting = false
	program.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{0}})
}

func generateDependencyGraph() (string, error) {
	clientset, err := getClientset()
	if err != nil {
		return "", fmt.Errorf("error getting clientset: %v", err)
	}

	viz := visualizer.New(clientset, namespace)
	outputPath := "dependency_graph.png"
	err = viz.CreateGraph(outputPath)
	if err != nil {
		return "", fmt.Errorf("error generating dependency graph: %v", err)
	}

	return fmt.Sprintf("Dependency graph generated and saved as %s", outputPath), nil
}

func analyzeResourcesForTUI() (string, error) {
	clientset, err := getClientset()
	if err != nil {
		return "", fmt.Errorf("error getting clientset: %v", err)
	}

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return "", fmt.Errorf("error getting config: %v", err)
	}

	metricsClientset, err := versioned.NewForConfig(config)
	if err != nil {
		return "", fmt.Errorf("error creating metrics clientset: %v", err)
	}

	pods, err := clientset.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return "", fmt.Errorf("error listing pods: %v", err)
	}

	podMetrics, err := metricsClientset.MetricsV1beta1().PodMetricses(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return "", fmt.Errorf("error getting pod metrics: %v", err)
	}

	var result strings.Builder
	w := tabwriter.NewWriter(&result, 0, 0, 2, ' ', 0)

	fmt.Fprintf(w, "POD\tCPU (cores)\tMEMORY (bytes)\tSTATUS\n")
	for _, pod := range pods.Items {
		cpuUsage := int64(0)
		memoryUsage := int64(0)

		for _, metric := range podMetrics.Items {
			if metric.Name == pod.Name {
				for _, container := range metric.Containers {
					cpuUsage += container.Usage.Cpu().MilliValue()
					memoryUsage += container.Usage.Memory().Value()
				}
				break
			}
		}

		fmt.Fprintf(w, "%s\t%dm\t%d\t%s\n", pod.Name, cpuUsage, memoryUsage, pod.Status.Phase)
	}

	w.Flush()
	return result.String(), nil
}

func checkClusterHealth() (string, error) {
	clientset, err := getClientset()
	if err != nil {
		return "", fmt.Errorf("error getting clientset: %v", err)
	}

	nodes, err := clientset.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return "", fmt.Errorf("error listing nodes: %v", err)
	}

	var result strings.Builder
	w := tabwriter.NewWriter(&result, 0, 0, 2, ' ', 0)

	fmt.Fprintf(w, "NODE\tSTATUS\tVERSION\n")
	for _, node := range nodes.Items {
		status := "Ready"
		for _, condition := range node.Status.Conditions {
			if condition.Type == corev1.NodeReady && condition.Status != corev1.ConditionTrue {
				status = "NotReady"
				break
			}
		}
		fmt.Fprintf(w, "%s\t%s\t%s\n", node.Name, status, node.Status.NodeInfo.KubeletVersion)
	}

	w.Flush()
	return result.String(), nil
}

func getClientset() (*kubernetes.Clientset, error) {
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("error building kubeconfig: %v", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("error creating clientset: %v", err)
	}

	return clientset, nil
}

func formatPodList(pods *corev1.PodList) string {
	var result strings.Builder
	w := tabwriter.NewWriter(&result, 0, 0, 2, ' ', 0)

	fmt.Fprintf(w, "NAMESPACE\tNAME\tSTATUS\tAGE\n")
	for _, pod := range pods.Items {
		age := time.Since(pod.CreationTimestamp.Time).Round(time.Second)
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", pod.Namespace, pod.Name, pod.Status.Phase, age)
	}

	w.Flush()
	return result.String()
}

func formatServiceAccountList(sas *corev1.ServiceAccountList) string {
	var result strings.Builder
	w := tabwriter.NewWriter(&result, 0, 0, 2, ' ', 0)

	fmt.Fprintf(w, "NAMESPACE\tNAME\tSECRETS\tAGE\n")
	for _, sa := range sas.Items {
		age := time.Since(sa.CreationTimestamp.Time).Round(time.Second)
		fmt.Fprintf(w, "%s\t%s\t%d\t%s\n", sa.Namespace, sa.Name, len(sa.Secrets), age)
	}

	w.Flush()
	return result.String()
}

func formatRolesList(roles *rbacv1.RoleList) string {
	var result strings.Builder
	w := tabwriter.NewWriter(&result, 0, 0, 2, ' ', 0)

	fmt.Fprintf(w, "NAMESPACE\tNAME\tAGE\n")
	for _, role := range roles.Items {
		age := time.Since(role.CreationTimestamp.Time).Round(time.Second)
		fmt.Fprintf(w, "%s\t%s\t%s\n", role.Namespace, role.Name, age)
	}

	w.Flush()
	return result.String()
}

// Additional helper functions

func deletePod(name, namespace string) error {
	clientset, err := getClientset()
	if err != nil {
		return fmt.Errorf("error getting clientset: %v", err)
	}

	err = clientset.CoreV1().Pods(namespace).Delete(context.TODO(), name, metav1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("error deleting pod: %v", err)
	}

	return nil
}

func getPodDetails(name, namespace string) (string, error) {
	clientset, err := getClientset()
	if err != nil {
		return "", fmt.Errorf("error getting clientset: %v", err)
	}

	pod, err := clientset.CoreV1().Pods(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("error getting pod details: %v", err)
	}

	var result strings.Builder
	w := tabwriter.NewWriter(&result, 0, 0, 2, ' ', 0)

	fmt.Fprintf(w, "Name:\t%s\n", pod.Name)
	fmt.Fprintf(w, "Namespace:\t%s\n", pod.Namespace)
	fmt.Fprintf(w, "Status:\t%s\n", pod.Status.Phase)
	fmt.Fprintf(w, "IP:\t%s\n", pod.Status.PodIP)
	fmt.Fprintf(w, "Node:\t%s\n", pod.Spec.NodeName)
	fmt.Fprintf(w, "Start Time:\t%s\n", pod.Status.StartTime.Time)

	fmt.Fprintf(w, "\nContainers:\n")
	for _, container := range pod.Spec.Containers {
		fmt.Fprintf(w, "  - %s:\n", container.Name)
		fmt.Fprintf(w, "    Image:\t%s\n", container.Image)
		fmt.Fprintf(w, "    Ready:\t%v\n", getContainerStatus(pod, container.Name))
	}

	w.Flush()
	return result.String(), nil
}

func getContainerStatus(pod *corev1.Pod, containerName string) bool {
	for _, status := range pod.Status.ContainerStatuses {
		if status.Name == containerName {
			return status.Ready
		}
	}
	return false
}

func getPodLogs(name, namespace string) (string, error) {
	clientset, err := getClientset()
	if err != nil {
		return "", fmt.Errorf("error getting clientset: %v", err)
	}

	podLogOpts := corev1.PodLogOptions{}
	req := clientset.CoreV1().Pods(namespace).GetLogs(name, &podLogOpts)
	podLogs, err := req.Stream(context.TODO())
	if err != nil {
		return "", fmt.Errorf("error in opening stream: %v", err)
	}
	defer podLogs.Close()

	buf := new(strings.Builder)
	_, err = io.Copy(buf, podLogs)
	if err != nil {
		return "", fmt.Errorf("error in copy information from podLogs to buf: %v", err)
	}

	return buf.String(), nil
}

func createServiceAccount(name, namespace string) error {
	clientset, err := getClientset()
	if err != nil {
		return fmt.Errorf("error getting clientset: %v", err)
	}

	sa := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}

	_, err = clientset.CoreV1().ServiceAccounts(namespace).Create(context.TODO(), sa, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("error creating service account: %v", err)
	}

	return nil
}

func deleteServiceAccount(name, namespace string) error {
	clientset, err := getClientset()
	if err != nil {
		return fmt.Errorf("error getting clientset: %v", err)
	}

	err = clientset.CoreV1().ServiceAccounts(namespace).Delete(context.TODO(), name, metav1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("error deleting service account: %v", err)
	}

	return nil
}

func createRole(name, namespace string, rules []rbacv1.PolicyRule) error {
	clientset, err := getClientset()
	if err != nil {
		return fmt.Errorf("error getting clientset: %v", err)
	}

	role := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Rules: rules,
	}

	_, err = clientset.RbacV1().Roles(namespace).Create(context.TODO(), role, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("error creating role: %v", err)
	}

	return nil
}

func deleteRole(name, namespace string) error {
	clientset, err := getClientset()
	if err != nil {
		return fmt.Errorf("error getting clientset: %v", err)
	}

	err = clientset.RbacV1().Roles(namespace).Delete(context.TODO(), name, metav1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("error deleting role: %v", err)
	}

	return nil
}
