package uix

import "strings"

type ModuleRegistry struct {
	modules map[string]Module
	aliases map[string]string
	order   []string
}

func NewModuleRegistry() *ModuleRegistry {
	return &ModuleRegistry{
		modules: make(map[string]Module),
		aliases: make(map[string]string),
		order:   make([]string, 0),
	}
}

func (r *ModuleRegistry) Register(module Module) {
	if module == nil {
		return
	}

	name := strings.TrimSpace(module.Name())
	if name == "" {
		return
	}
	if _, exists := r.modules[name]; !exists {
		r.order = append(r.order, name)
	}
	r.modules[name] = module

	for _, alias := range module.Aliases() {
		alias = strings.TrimSpace(alias)
		if alias != "" {
			r.aliases[alias] = name
		}
	}
}

func (r *ModuleRegistry) Resolve(input string) Module {
	input = strings.TrimPrefix(strings.TrimSpace(input), "/")
	if input == "" {
		return nil
	}
	if module, ok := r.modules[input]; ok {
		return module
	}
	if name, ok := r.aliases[input]; ok {
		return r.modules[name]
	}
	return nil
}

func (r *ModuleRegistry) Search(query string) []Module {
	query = strings.ToLower(strings.TrimPrefix(strings.TrimSpace(query), "/"))
	if query == "" {
		return r.List()
	}

	result := make([]Module, 0)
	for _, name := range r.order {
		module := r.modules[name]
		if matchesModule(module, query) {
			result = append(result, module)
		}
	}
	return result
}

func (r *ModuleRegistry) List() []Module {
	modules := make([]Module, 0, len(r.order))
	for _, name := range r.order {
		modules = append(modules, r.modules[name])
	}
	return modules
}

func matchesModule(module Module, query string) bool {
	if strings.Contains(strings.ToLower(module.Name()), query) {
		return true
	}
	if strings.Contains(strings.ToLower(module.Description()), query) {
		return true
	}
	for _, alias := range module.Aliases() {
		if strings.Contains(strings.ToLower(alias), query) {
			return true
		}
	}
	return false
}

type CommandRegistry struct {
	commands map[string]Command
	aliases  map[string]string
	order    []string
}

func NewCommandRegistry() *CommandRegistry {
	return &CommandRegistry{
		commands: make(map[string]Command),
		aliases:  make(map[string]string),
		order:    make([]string, 0),
	}
}

func (r *CommandRegistry) Register(command Command) {
	name := strings.TrimSpace(command.Name)
	if name == "" || command.Run == nil {
		return
	}
	command.Name = name
	if _, exists := r.commands[name]; !exists {
		r.order = append(r.order, name)
	}
	r.commands[name] = command

	for _, alias := range command.Aliases {
		alias = strings.TrimSpace(alias)
		if alias != "" {
			r.aliases[alias] = name
		}
	}
}

func (r *CommandRegistry) Resolve(input string) *Command {
	input = strings.TrimPrefix(strings.TrimSpace(input), "/")
	if input == "" {
		return nil
	}
	if command, ok := r.commands[input]; ok {
		return &command
	}
	if name, ok := r.aliases[input]; ok {
		command := r.commands[name]
		return &command
	}
	return nil
}

func (r *CommandRegistry) List() []Command {
	commands := make([]Command, 0, len(r.order))
	for _, name := range r.order {
		commands = append(commands, r.commands[name])
	}
	return commands
}

func (r *CommandRegistry) Search(query string) []Command {
	query = strings.ToLower(strings.TrimPrefix(strings.TrimSpace(query), "/"))
	if query == "" {
		return r.List()
	}

	result := make([]Command, 0)
	for _, name := range r.order {
		command := r.commands[name]
		if matchesCommand(command, query) {
			result = append(result, command)
		}
	}
	return result
}

func matchesCommand(command Command, query string) bool {
	if strings.Contains(strings.ToLower(command.Name), query) {
		return true
	}
	if strings.Contains(strings.ToLower(command.Description), query) {
		return true
	}
	for _, alias := range command.Aliases {
		if strings.Contains(strings.ToLower(alias), query) {
			return true
		}
	}
	return false
}
