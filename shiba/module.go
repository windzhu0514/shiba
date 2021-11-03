package shiba

import (
	"reflect"
	"sort"
)

type Module interface {
	Name() string // 模块名
	Init() error  // 初始化模块（添加路由、flags）
	Start() error // 启动模块（启动模块功能）
	Stop() error  // 停止模块
}

type module struct {
	Name     string
	Priority int
	Module   Module
}

var modules = make([]module, 0)

func registerModule(priority int, mod Module) {
	name := mod.Name()
	ok := isExist(name)
	if ok {
		panic("Module " + name + " is alreadly registered")
	}

	rv := reflect.ValueOf(mod)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		panic("Module " + name + " is non-pointer or nil")
	}

	modules = append(modules, module{
		Name:     name,
		Priority: priority,
		Module:   mod,
	})

	sort.SliceStable(modules, func(i, j int) bool {
		return modules[i].Priority < modules[j].Priority
	})
}

func getModule(name string) Module {
	for _, mod := range modules {
		if mod.Name == name {
			return mod.Module
		}
	}

	return nil
}

func isExist(name string) bool {
	for _, mod := range modules {
		if mod.Name == name {
			return true
		}
	}

	return false
}
