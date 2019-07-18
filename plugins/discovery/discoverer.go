package discovery

import (
	"fmt"
	"github.com/golang/glog"
	"github.com/wavefronthq/wavefront-kubernetes-collector/internal/discovery"
	"github.com/wavefronthq/wavefront-kubernetes-collector/internal/metrics"
	"github.com/wavefronthq/wavefront-kubernetes-collector/plugins/discovery/prometheus"
	"github.com/wavefronthq/wavefront-kubernetes-collector/plugins/discovery/telegraf"
	"strings"
	"sync"
)

type delegate struct {
	filter  *resourceFilter
	handler discovery.TargetHandler
	plugin  discovery.PluginConfig
}

type discoverer struct {
	queue           chan discovery.Resource
	runtimeHandlers []discovery.TargetHandler
	mtx             sync.RWMutex
	delegates       map[string]*delegate
}

func newDiscoverer(handler metrics.ProviderHandler, plugins []discovery.PluginConfig) discovery.Discoverer {
	d := &discoverer{
		queue:           make(chan discovery.Resource, 1000),
		runtimeHandlers: makeRuntimeHandlers(handler),
		delegates:       makeDelegates(handler, plugins),
	}
	go d.process()
	return d
}

func makeRuntimeHandlers(handler metrics.ProviderHandler) []discovery.TargetHandler {
	// currently annotation based discovery is supported only for prometheus
	return []discovery.TargetHandler{
		prometheus.NewTargetHandler(handler, true),
	}
}

func makeDelegates(handler metrics.ProviderHandler, plugins []discovery.PluginConfig) map[string]*delegate {
	delegates := make(map[string]*delegate, len(plugins))
	for _, plugin := range plugins {
		delegate, err := makeDelegate(handler, plugin)
		if err != nil {
			glog.Errorf("error parsing plugin: %s error: %v", plugin.Name, err)
			continue
		}
		delegates[plugin.Name] = delegate
	}
	return delegates
}

func makeDelegate(handler metrics.ProviderHandler, plugin discovery.PluginConfig) (*delegate, error) {
	filter, err := newResourceFilter(plugin)
	if err != nil {
		return nil, err
	}
	var targetHandler discovery.TargetHandler
	if strings.Contains(plugin.Type, "prometheus") {
		targetHandler = prometheus.NewTargetHandler(handler, false)
	} else if strings.Contains(plugin.Type, "telegraf") {
		targetHandler = telegraf.NewTargetHandler(handler, plugin.Type)
	} else {
		return nil, fmt.Errorf("invalid plugin type: %s", plugin.Type)
	}
	return &delegate{
		handler: targetHandler,
		filter:  filter,
		plugin:  plugin,
	}, nil
}

func (d *discoverer) process() {
	for {
		select {
		case resource, more := <-d.queue:
			if !more {
				glog.Infof("stopping resource discovery processing")
				return
			}
			switch resource.Status {
			case "delete":
				d.internalDelete(resource)
			default:
				d.internalDiscover(resource)
			}
		}
	}
}

func (d *discoverer) Stop() {
	close(d.queue)
}

func (d *discoverer) Discover(resource discovery.Resource) {
	d.queue <- resource
}

func (d *discoverer) Delete(resource discovery.Resource) {
	resource.Status = "delete"
	d.queue <- resource
}

func (d *discoverer) internalDiscover(resource discovery.Resource) {
	d.mtx.RLock()
	defer d.mtx.RUnlock()

	for _, delegate := range d.delegates {
		if delegate.filter.matches(resource) {
			delegate.handler.Handle(resource, delegate.plugin)
			return
		}
	}
	// delegate to runtime handlers if no matching delegate
	for _, runtimeHandler := range d.runtimeHandlers {
		runtimeHandler.Handle(resource, nil)
	}
}

func (d *discoverer) internalDelete(resource discovery.Resource) {
	d.mtx.RLock()
	defer d.mtx.RUnlock()

	name := discovery.ResourceName(resource.Kind, resource.Meta)
	for _, delegate := range d.delegates {
		if delegate.filter.matches(resource) {
			delegate.handler.Delete(name)
			return
		}
	}
	// delegate to runtime handlers if no matching delegate
	for _, runtimeHandler := range d.runtimeHandlers {
		runtimeHandler.Delete(name)
	}
}
