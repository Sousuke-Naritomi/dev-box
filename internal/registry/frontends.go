package registry

import "github.com/Sousuke-Naritomi/dev-box/internal/config"

type FrontendFramework struct {
	ID          string
	Name        string
	Image       string
	Command     string
	Volumes     []string
	TemplateDir string
}

var SupportedFrontends = map[string]FrontendFramework{
	"react": {
		ID:          "react",
		Name:        "React (Vite)",
		Image:       "node:" + config.VersionNode,
		Command:     `sh -c "npm install && npm run dev -- --host 0.0.0.0"`,
		Volumes:     []string{"./frontend:/app", "frontend-node-modules:/app/node_modules"},
		TemplateDir: "templates/scaffold/frontend/react",
	},
	"vue": {
		ID:          "vue",
		Name:        "Vue.js (Vite)",
		Image:       "node:" + config.VersionNode,
		Command:     `sh -c "npm install && npm run dev -- --host 0.0.0.0"`,
		Volumes:     []string{"./frontend:/app", "frontend-node-modules:/app/node_modules"},
		TemplateDir: "templates/scaffold/frontend/vue",
	},
}
