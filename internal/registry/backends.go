package registry

import "github.com/Sousuke-Naritomi/dev-box/internal/config"

type BackendFramework struct {
	ID             string
	Name           string
	Image          string
	InternalPort   int
	Command        string
	Volumes        []string
	TemplateDir    string
}

var SupportedBackends = map[string]BackendFramework{
	"node": {
		ID:           "node",
		Name:         "Node.js (Express)",
		Image:        "node:" + config.VersionNode,
		InternalPort: 3000,
		Command:      `sh -c "npm install && npm start"`,
		Volumes:      []string{"./backend:/app", "backend-node-modules:/app/node_modules"},
		TemplateDir:  "templates/scaffold/backend/node",
	},
	"python": {
		ID:           "python",
		Name:         "Python (Flask)",
		Image:        "python:" + config.VersionPython,
		InternalPort: 8000,
		Command:      `sh -c "pip install -r requirements.txt && python main.py"`,
		Volumes:      []string{"./backend:/app"},
		TemplateDir:  "templates/scaffold/backend/python",
	},
	"go": {
		ID:           "go",
		Name:         "Go (net/http)",
		Image:        "golang:" + config.VersionGo,
		InternalPort: 8080,
		Command:      `sh -c "go run main.go"`,
		Volumes:      []string{"./backend:/app", "go-mod-cache:/go/pkg/mod"},
		TemplateDir:  "templates/scaffold/backend/go",
	},
	"php": {
		ID:           "php",
		Name:         "PHP (Apache)",
		Image:        "php:" + config.VersionPHP,
		InternalPort: 80,
		Command:      "",
		Volumes:      []string{"./backend:/var/www/html"},
		TemplateDir:  "templates/scaffold/backend/php",
	},
	"java": {
		ID:           "java",
		Name:         "Java (HttpServer)",
		Image:        "eclipse-temurin:" + config.VersionJava,
		InternalPort: 8080,
		Command:      `sh -c "javac Main.java && java Main"`,
		Volumes:      []string{"./backend:/app"},
		TemplateDir:  "templates/scaffold/backend/java",
	},
}
