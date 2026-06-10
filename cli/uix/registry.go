package uix

import "strings"

type PluginRegistry struct {
	plugins map[string]Plugin
	aliases map[string]string
	order   []string
}

func NewRegistry() *PluginRegistry {
	return &PluginRegistry{
		plugins: make(map[string]Plugin),
		aliases: make(map[string]string),
		order:   make([]string, 0),
	}
}

func (r *PluginRegistry) Register(p Plugin) {
	if p == nil {
		return
	}

	name := strings.TrimSpace(p.Name())
	if name == "" {
		return
	}
	if _, exists := r.plugins[name]; !exists {
		r.order = append(r.order, name)
	}
	r.plugins[name] = p

	for _, alias := range p.Aliases() {
		alias = strings.TrimSpace(alias)
		if alias != "" {
			r.aliases[alias] = name
		}
	}
}

func (r *PluginRegistry) Resolve(input string) Plugin {
	input = strings.TrimPrefix(strings.TrimSpace(input), "/")
	if input == "" {
		return nil
	}
	if plugin, ok := r.plugins[input]; ok {
		return plugin
	}
	if name, ok := r.aliases[input]; ok {
		return r.plugins[name]
	}
	return nil
}

func (r *PluginRegistry) Search(query string) []Plugin {
	query = strings.ToLower(strings.TrimPrefix(strings.TrimSpace(query), "/"))
	if query == "" {
		return r.List()
	}

	result := make([]Plugin, 0)
	for _, name := range r.order {
		plugin := r.plugins[name]
		if matchesPlugin(plugin, query) {
			result = append(result, plugin)
		}
	}
	return result
}

func (r *PluginRegistry) List() []Plugin {
	plugins := make([]Plugin, 0, len(r.order))
	for _, name := range r.order {
		plugins = append(plugins, r.plugins[name])
	}
	return plugins
}

func matchesPlugin(plugin Plugin, query string) bool {
	if strings.Contains(strings.ToLower(plugin.Name()), query) {
		return true
	}
	if strings.Contains(strings.ToLower(plugin.Description()), query) {
		return true
	}
	for _, alias := range plugin.Aliases() {
		if strings.Contains(strings.ToLower(alias), query) {
			return true
		}
	}
	return false
}
