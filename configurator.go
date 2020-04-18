// Package configuration provides ability to initialize your custom configuration struct from: flags, environment variables, `default` tag, files (json, yaml)
package configuration

import (
	"errors"
	"log"
	"reflect"
)

// New creates a new instance of the configurator.
// 'gLoggingEnabled' and 'gFailIfCannotSet' both are set to 'true' by default
// default logger function is set to `log.Printf`
func New(
	cfgPtr interface{}, // must be a pointer to a struct
	providers ...Provider, // providers will be executed in order of their declaration
) (configurator, error) {
	if len(providers) == 0 {
		return configurator{}, errors.New("providers not found")
	}

	if reflect.TypeOf(cfgPtr).Kind() != reflect.Ptr {
		return configurator{}, errors.New("not a pointer to the struct")
	}

	gLoggingEnabled = true
	gFailIfCannotSet = true
	logger = log.Printf

	return configurator{
		config:    cfgPtr,
		providers: providers,
	}, nil
}

type configurator struct {
	config    interface{}
	providers []Provider
}

// InitValues sets values into struct field using given set of providers
// respecting their order: first defined -> first executed
func (c configurator) InitValues() {
	c.fillUp(c.config)
}

// SetLogger changes logger
func (c configurator) SetLogger(l Logger) configurator {
	logger = l
	return c
}

// DisableLogging turns off logging
func (c configurator) DisableLogging() configurator {
	gLoggingEnabled = false
	return c
}

// IgnoreErrors prevents calling os.Exit(1) if the lib fails to init a field
func (c configurator) IgnoreErrors() configurator {
	gFailIfCannotSet = false
	return c
}

func (c configurator) fillUp(i interface{}, parentPath ...string) {
	var (
		t = reflect.TypeOf(i)
		v = reflect.ValueOf(i)
	)

	if t.Kind() == reflect.Ptr {
		t = t.Elem()
		v = v.Elem()
	}

	for i := 0; i < t.NumField(); i++ {
		var (
			tField      = t.Field(i)
			vField      = v.Field(i)
			currentPath = append(parentPath, tField.Name)
		)

		if tField.Type.Kind() == reflect.Struct {
			c.fillUp(vField.Addr().Interface(), currentPath...)
			continue
		}

		if tField.Type.Kind() == reflect.Ptr && tField.Type.Elem().Kind() == reflect.Struct {
			vField.Set(reflect.New(tField.Type.Elem()))
			c.fillUp(vField.Interface(), currentPath...)
			continue
		}

		c.applyProviders(tField, vField, currentPath)
	}
}

func (c configurator) applyProviders(field reflect.StructField, v reflect.Value, currentPath []string) {
	logf("configurator: current path: %v", currentPath)

	for _, provider := range c.providers {
		if provider.Provide(field, v, currentPath...) {
			logf("\n")
			return
		}
	}
	logf("configurator: field [%s] with tags [%v] cannot be set!", field.Name, field.Tag)
	fatalf("configurator: field [%s] with tags [%v] cannot be set!", field.Name, field.Tag)
}
