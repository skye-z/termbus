package eventbus

import (
	"reflect"
	"sync"
)

// HandlerFunc represents an event handler function
type HandlerFunc func(...interface{})

// EventBus simple event bus implementation
type EventBus struct {
	mu     sync.RWMutex
	topics map[string][]HandlerFunc
}

// New creates a new event bus
func New() *EventBus {
	return &EventBus{
		topics: make(map[string][]HandlerFunc),
	}
}

// Subscribe subscribes a handler to a topic
func (eb *EventBus) Subscribe(topic string, handler interface{}) {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	handlerFunc := toHandlerFunc(handler)
	if handlerFunc != nil {
		eb.topics[topic] = append(eb.topics[topic], handlerFunc)
	}
}

// Unsubscribe removes a handler from a topic
func (eb *EventBus) Unsubscribe(topic string, handler interface{}) {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	handlers := eb.topics[topic]
	if handlers == nil {
		return
	}

	handlerFunc := toHandlerFunc(handler)
	if handlerFunc == nil {
		return
	}

	for i, h := range handlers {
		if reflect.ValueOf(h).Pointer() == reflect.ValueOf(handlerFunc).Pointer() {
			eb.topics[topic] = append(handlers[:i], handlers[i+1:]...)
			break
		}
	}
}

// Publish publishes an event to a topic
func (eb *EventBus) Publish(topic string, args ...interface{}) {
	eb.mu.RLock()
	handlers := eb.topics[topic]
	eb.mu.RUnlock()

	for _, handler := range handlers {
		go handler(args...)
	}
}

// HasCallback checks if a topic has any subscribers
func (eb *EventBus) HasCallback(topic string) bool {
	eb.mu.RLock()
	defer eb.mu.RUnlock()

	handlers, exists := eb.topics[topic]
	return exists && len(handlers) > 0
}

// toHandlerFunc converts various handler types to HandlerFunc
func toHandlerFunc(handler interface{}) HandlerFunc {
	if handler == nil {
		return nil
	}

	v := reflect.ValueOf(handler)
	if v.Kind() != reflect.Func {
		return nil
	}

	return func(args ...interface{}) {
		in := make([]reflect.Value, v.Type().NumIn())
		for i := range in {
			if i < len(args) && args[i] != nil {
				in[i] = reflect.ValueOf(args[i])
			} else {
				in[i] = reflect.Zero(v.Type().In(i))
			}
		}
		v.Call(in)
	}
}

// Manager 事件总线管理器
type Manager struct {
	bus *EventBus
}

// NewManager 创建事件总线管理器
func NewManager() *Manager {
	return &Manager{
		bus: New(),
	}
}

// Subscribe 订阅事件
func (m *Manager) Subscribe(topic string, handler interface{}) {
	m.bus.Subscribe(topic, handler)
}

// Unsubscribe 取消订阅
func (m *Manager) Unsubscribe(topic string, handler interface{}) {
	m.bus.Unsubscribe(topic, handler)
}

// Publish 发布事件
func (m *Manager) Publish(topic string, args ...interface{}) {
	m.bus.Publish(topic, args...)
}

// HasTopic 检查主题是否存在
func (m *Manager) HasTopic(topic string) bool {
	return m.bus.HasCallback(topic)
}
