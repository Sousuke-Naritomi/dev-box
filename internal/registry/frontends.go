package registry

import "github.com/Sousuke-Naritomi/dev-box/internal/config"

type FrontendFramework struct {
	ID           string
	Name         string
	Image        string
	InternalPort int
	Command      string
	Volumes      []string
	TemplateDir  string
}

var SupportedFrontends = map[config.FrontendType]FrontendFramework{
	config.FrontendReact: {
		ID:           "react",
		Name:         "React (Vite)",
		Image:        "node:" + config.VersionNode,
		InternalPort: 5173,
		Command:      `sh -c "npm install && npm run dev -- --host 0.0.0.0"`,
		Volumes:      []string{"./frontend:/app", "frontend-node-modules:/app/node_modules"},
		TemplateDir:  "templates/scaffold/frontend/react",
	},
	config.FrontendVue: {
		ID:           "vue",
		Name:         "Vue.js (Vite)",
		Image:        "node:" + config.VersionNode,
		InternalPort: 5173,
		Command:      `sh -c "npm install && npm run dev -- --host 0.0.0.0"`,
		Volumes:      []string{"./frontend:/app", "frontend-node-modules:/app/node_modules"},
		TemplateDir:  "templates/scaffold/frontend/vue",
	},
}
