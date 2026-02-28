package server

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
)

type appMeshStore struct {
	mu sync.Mutex

	meshes          map[string]map[string]any
	virtualGateways map[string]map[string]map[string]any
	gatewayRoutes   map[string]map[string]map[string]map[string]any
	virtualNodes    map[string]map[string]map[string]any
	virtualRouters  map[string]map[string]map[string]any
	routes          map[string]map[string]map[string]map[string]any
	virtualServices map[string]map[string]map[string]any
	tags            map[string]map[string]string
}

func newAppMeshStore() *appMeshStore {
	s := &appMeshStore{
		meshes:          map[string]map[string]any{},
		virtualGateways: map[string]map[string]map[string]any{},
		gatewayRoutes:   map[string]map[string]map[string]map[string]any{},
		virtualNodes:    map[string]map[string]map[string]any{},
		virtualRouters:  map[string]map[string]map[string]any{},
		routes:          map[string]map[string]map[string]map[string]any{},
		virtualServices: map[string]map[string]map[string]any{},
		tags:            map[string]map[string]string{},
	}

	now := time.Now().UTC().Format(time.RFC3339)
	mesh := s.newMesh("apps", now)
	s.meshes["apps"] = mesh

	vg := s.newVirtualGateway("apps", "stackyard-gateway", now)
	s.ensureVirtualGatewayMapLocked("apps")["stackyard-gateway"] = vg
	gr := s.newGatewayRoute("apps", "stackyard-gateway", "stackyard-gateway-route", now)
	s.ensureGatewayRouteMapLocked("apps", "stackyard-gateway")["stackyard-gateway-route"] = gr

	vn := s.newVirtualNode("apps", "stackyard-node", now)
	s.ensureVirtualNodeMapLocked("apps")["stackyard-node"] = vn

	vr := s.newVirtualRouter("apps", "stackyard-router", now)
	s.ensureVirtualRouterMapLocked("apps")["stackyard-router"] = vr
	rt := s.newRoute("apps", "stackyard-router", "stackyard-route", now)
	s.ensureRouteMapLocked("apps", "stackyard-router")["stackyard-route"] = rt

	vs := s.newVirtualService("apps", "stackyard.local", now)
	s.ensureVirtualServiceMapLocked("apps")["stackyard.local"] = vs

	s.tags[appMeshMeshARN("apps")] = map[string]string{"seed": "true"}
	return s
}

func (s *appMeshStore) Handle(action string, payload map[string]any, pathParams map[string]string, query url.Values) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC().Format(time.RFC3339)

	s.syncPayloadWithQuery(payload, query)

	switch action {
	case "CreateMesh":
		meshName := appMeshFirstNonEmpty(
			appMeshString(pathParams, "meshName", ""),
			appMeshString(payload, "meshName", ""),
			"apps",
		)
		mesh := s.ensureMeshLocked(meshName, now)
		appMeshMergeSpec(mesh, payload)
		appMeshTouchMetadata(mesh, now)
		return map[string]any{"mesh": appMeshCloneMap(mesh)}

	case "DeleteMesh":
		meshName := appMeshFirstNonEmpty(appMeshString(pathParams, "meshName", ""), appMeshString(payload, "meshName", ""), "apps")
		mesh := s.ensureMeshLocked(meshName, now)
		appMeshSetStatus(mesh, "DELETED")
		appMeshTouchMetadata(mesh, now)
		delete(s.meshes, meshName)
		delete(s.virtualGateways, meshName)
		delete(s.gatewayRoutes, meshName)
		delete(s.virtualNodes, meshName)
		delete(s.virtualRouters, meshName)
		delete(s.routes, meshName)
		delete(s.virtualServices, meshName)
		return map[string]any{"mesh": appMeshCloneMap(mesh)}

	case "DescribeMesh":
		meshName := appMeshFirstNonEmpty(appMeshString(pathParams, "meshName", ""), appMeshString(payload, "meshName", ""), "apps")
		mesh := s.ensureMeshLocked(meshName, now)
		return map[string]any{"mesh": appMeshCloneMap(mesh)}

	case "ListMeshes":
		return map[string]any{"meshes": appMeshListMeshes(s.meshes), "nextToken": ""}

	case "UpdateMesh":
		meshName := appMeshFirstNonEmpty(appMeshString(pathParams, "meshName", ""), appMeshString(payload, "meshName", ""), "apps")
		mesh := s.ensureMeshLocked(meshName, now)
		appMeshMergeSpec(mesh, payload)
		appMeshTouchMetadata(mesh, now)
		return map[string]any{"mesh": appMeshCloneMap(mesh)}

	case "CreateVirtualGateway":
		meshName := appMeshFirstNonEmpty(appMeshString(pathParams, "meshName", ""), appMeshString(payload, "meshName", ""), "apps")
		name := appMeshFirstNonEmpty(appMeshString(payload, "virtualGatewayName", ""), "stackyard-virtual-gateway")
		vg := s.ensureVirtualGatewayLocked(meshName, name, now)
		appMeshMergeSpec(vg, payload)
		appMeshTouchMetadata(vg, now)
		return map[string]any{"virtualGateway": appMeshCloneMap(vg)}

	case "DeleteVirtualGateway":
		meshName := appMeshFirstNonEmpty(appMeshString(pathParams, "meshName", ""), "apps")
		name := appMeshFirstNonEmpty(appMeshString(pathParams, "virtualGatewayName", ""), appMeshString(payload, "virtualGatewayName", ""), "stackyard-virtual-gateway")
		vg := s.ensureVirtualGatewayLocked(meshName, name, now)
		appMeshSetStatus(vg, "DELETED")
		appMeshTouchMetadata(vg, now)
		delete(s.ensureVirtualGatewayMapLocked(meshName), name)
		return map[string]any{"virtualGateway": appMeshCloneMap(vg)}

	case "DescribeVirtualGateway":
		meshName := appMeshFirstNonEmpty(appMeshString(pathParams, "meshName", ""), "apps")
		name := appMeshFirstNonEmpty(appMeshString(pathParams, "virtualGatewayName", ""), appMeshString(payload, "virtualGatewayName", ""), "stackyard-virtual-gateway")
		vg := s.ensureVirtualGatewayLocked(meshName, name, now)
		return map[string]any{"virtualGateway": appMeshCloneMap(vg)}

	case "ListVirtualGateways":
		meshName := appMeshFirstNonEmpty(appMeshString(pathParams, "meshName", ""), appMeshString(payload, "meshName", ""), "apps")
		return map[string]any{"virtualGateways": appMeshListNamed(s.ensureVirtualGatewayMapLocked(meshName), "virtualGatewayName"), "nextToken": ""}

	case "UpdateVirtualGateway":
		meshName := appMeshFirstNonEmpty(appMeshString(pathParams, "meshName", ""), "apps")
		name := appMeshFirstNonEmpty(appMeshString(pathParams, "virtualGatewayName", ""), appMeshString(payload, "virtualGatewayName", ""), "stackyard-virtual-gateway")
		vg := s.ensureVirtualGatewayLocked(meshName, name, now)
		appMeshMergeSpec(vg, payload)
		appMeshTouchMetadata(vg, now)
		return map[string]any{"virtualGateway": appMeshCloneMap(vg)}

	case "CreateGatewayRoute":
		meshName := appMeshFirstNonEmpty(appMeshString(pathParams, "meshName", ""), "apps")
		virtualGatewayName := appMeshFirstNonEmpty(appMeshString(pathParams, "virtualGatewayName", ""), appMeshString(payload, "virtualGatewayName", ""), "stackyard-virtual-gateway")
		name := appMeshFirstNonEmpty(appMeshString(payload, "gatewayRouteName", ""), "stackyard-gateway-route")
		gr := s.ensureGatewayRouteLocked(meshName, virtualGatewayName, name, now)
		appMeshMergeSpec(gr, payload)
		appMeshTouchMetadata(gr, now)
		return map[string]any{"gatewayRoute": appMeshCloneMap(gr)}

	case "DeleteGatewayRoute":
		meshName := appMeshFirstNonEmpty(appMeshString(pathParams, "meshName", ""), "apps")
		virtualGatewayName := appMeshFirstNonEmpty(appMeshString(pathParams, "virtualGatewayName", ""), "stackyard-virtual-gateway")
		name := appMeshFirstNonEmpty(appMeshString(pathParams, "gatewayRouteName", ""), appMeshString(payload, "gatewayRouteName", ""), "stackyard-gateway-route")
		gr := s.ensureGatewayRouteLocked(meshName, virtualGatewayName, name, now)
		appMeshSetStatus(gr, "DELETED")
		appMeshTouchMetadata(gr, now)
		delete(s.ensureGatewayRouteMapLocked(meshName, virtualGatewayName), name)
		return map[string]any{"gatewayRoute": appMeshCloneMap(gr)}

	case "DescribeGatewayRoute":
		meshName := appMeshFirstNonEmpty(appMeshString(pathParams, "meshName", ""), "apps")
		virtualGatewayName := appMeshFirstNonEmpty(appMeshString(pathParams, "virtualGatewayName", ""), "stackyard-virtual-gateway")
		name := appMeshFirstNonEmpty(appMeshString(pathParams, "gatewayRouteName", ""), appMeshString(payload, "gatewayRouteName", ""), "stackyard-gateway-route")
		gr := s.ensureGatewayRouteLocked(meshName, virtualGatewayName, name, now)
		return map[string]any{"gatewayRoute": appMeshCloneMap(gr)}

	case "ListGatewayRoutes":
		meshName := appMeshFirstNonEmpty(appMeshString(pathParams, "meshName", ""), appMeshString(payload, "meshName", ""), "apps")
		virtualGatewayName := appMeshFirstNonEmpty(appMeshString(pathParams, "virtualGatewayName", ""), appMeshString(payload, "virtualGatewayName", ""), "stackyard-virtual-gateway")
		return map[string]any{"gatewayRoutes": appMeshListNamed(s.ensureGatewayRouteMapLocked(meshName, virtualGatewayName), "gatewayRouteName"), "nextToken": ""}

	case "UpdateGatewayRoute":
		meshName := appMeshFirstNonEmpty(appMeshString(pathParams, "meshName", ""), "apps")
		virtualGatewayName := appMeshFirstNonEmpty(appMeshString(pathParams, "virtualGatewayName", ""), "stackyard-virtual-gateway")
		name := appMeshFirstNonEmpty(appMeshString(pathParams, "gatewayRouteName", ""), appMeshString(payload, "gatewayRouteName", ""), "stackyard-gateway-route")
		gr := s.ensureGatewayRouteLocked(meshName, virtualGatewayName, name, now)
		appMeshMergeSpec(gr, payload)
		appMeshTouchMetadata(gr, now)
		return map[string]any{"gatewayRoute": appMeshCloneMap(gr)}

	case "CreateVirtualNode":
		meshName := appMeshFirstNonEmpty(appMeshString(pathParams, "meshName", ""), appMeshString(payload, "meshName", ""), "apps")
		name := appMeshFirstNonEmpty(appMeshString(payload, "virtualNodeName", ""), "stackyard-virtual-node")
		vn := s.ensureVirtualNodeLocked(meshName, name, now)
		appMeshMergeSpec(vn, payload)
		appMeshTouchMetadata(vn, now)
		return map[string]any{"virtualNode": appMeshCloneMap(vn)}

	case "DeleteVirtualNode":
		meshName := appMeshFirstNonEmpty(appMeshString(pathParams, "meshName", ""), "apps")
		name := appMeshFirstNonEmpty(appMeshString(pathParams, "virtualNodeName", ""), appMeshString(payload, "virtualNodeName", ""), "stackyard-virtual-node")
		vn := s.ensureVirtualNodeLocked(meshName, name, now)
		appMeshSetStatus(vn, "DELETED")
		appMeshTouchMetadata(vn, now)
		delete(s.ensureVirtualNodeMapLocked(meshName), name)
		return map[string]any{"virtualNode": appMeshCloneMap(vn)}

	case "DescribeVirtualNode":
		meshName := appMeshFirstNonEmpty(appMeshString(pathParams, "meshName", ""), "apps")
		name := appMeshFirstNonEmpty(appMeshString(pathParams, "virtualNodeName", ""), appMeshString(payload, "virtualNodeName", ""), "stackyard-virtual-node")
		vn := s.ensureVirtualNodeLocked(meshName, name, now)
		return map[string]any{"virtualNode": appMeshCloneMap(vn)}

	case "ListVirtualNodes":
		meshName := appMeshFirstNonEmpty(appMeshString(pathParams, "meshName", ""), appMeshString(payload, "meshName", ""), "apps")
		return map[string]any{"virtualNodes": appMeshListNamed(s.ensureVirtualNodeMapLocked(meshName), "virtualNodeName"), "nextToken": ""}

	case "UpdateVirtualNode":
		meshName := appMeshFirstNonEmpty(appMeshString(pathParams, "meshName", ""), "apps")
		name := appMeshFirstNonEmpty(appMeshString(pathParams, "virtualNodeName", ""), appMeshString(payload, "virtualNodeName", ""), "stackyard-virtual-node")
		vn := s.ensureVirtualNodeLocked(meshName, name, now)
		appMeshMergeSpec(vn, payload)
		appMeshTouchMetadata(vn, now)
		return map[string]any{"virtualNode": appMeshCloneMap(vn)}

	case "CreateVirtualRouter":
		meshName := appMeshFirstNonEmpty(appMeshString(pathParams, "meshName", ""), appMeshString(payload, "meshName", ""), "apps")
		name := appMeshFirstNonEmpty(appMeshString(payload, "virtualRouterName", ""), "stackyard-virtual-router")
		vr := s.ensureVirtualRouterLocked(meshName, name, now)
		appMeshMergeSpec(vr, payload)
		appMeshTouchMetadata(vr, now)
		return map[string]any{"virtualRouter": appMeshCloneMap(vr)}

	case "DeleteVirtualRouter":
		meshName := appMeshFirstNonEmpty(appMeshString(pathParams, "meshName", ""), "apps")
		name := appMeshFirstNonEmpty(appMeshString(pathParams, "virtualRouterName", ""), appMeshString(payload, "virtualRouterName", ""), "stackyard-virtual-router")
		vr := s.ensureVirtualRouterLocked(meshName, name, now)
		appMeshSetStatus(vr, "DELETED")
		appMeshTouchMetadata(vr, now)
		delete(s.ensureVirtualRouterMapLocked(meshName), name)
		delete(s.ensureRouteRouterMapLocked(meshName), name)
		return map[string]any{"virtualRouter": appMeshCloneMap(vr)}

	case "DescribeVirtualRouter":
		meshName := appMeshFirstNonEmpty(appMeshString(pathParams, "meshName", ""), "apps")
		name := appMeshFirstNonEmpty(appMeshString(pathParams, "virtualRouterName", ""), appMeshString(payload, "virtualRouterName", ""), "stackyard-virtual-router")
		vr := s.ensureVirtualRouterLocked(meshName, name, now)
		return map[string]any{"virtualRouter": appMeshCloneMap(vr)}

	case "ListVirtualRouters":
		meshName := appMeshFirstNonEmpty(appMeshString(pathParams, "meshName", ""), appMeshString(payload, "meshName", ""), "apps")
		return map[string]any{"virtualRouters": appMeshListNamed(s.ensureVirtualRouterMapLocked(meshName), "virtualRouterName"), "nextToken": ""}

	case "UpdateVirtualRouter":
		meshName := appMeshFirstNonEmpty(appMeshString(pathParams, "meshName", ""), "apps")
		name := appMeshFirstNonEmpty(appMeshString(pathParams, "virtualRouterName", ""), appMeshString(payload, "virtualRouterName", ""), "stackyard-virtual-router")
		vr := s.ensureVirtualRouterLocked(meshName, name, now)
		appMeshMergeSpec(vr, payload)
		appMeshTouchMetadata(vr, now)
		return map[string]any{"virtualRouter": appMeshCloneMap(vr)}

	case "CreateRoute":
		meshName := appMeshFirstNonEmpty(appMeshString(pathParams, "meshName", ""), "apps")
		virtualRouterName := appMeshFirstNonEmpty(appMeshString(pathParams, "virtualRouterName", ""), appMeshString(payload, "virtualRouterName", ""), "stackyard-virtual-router")
		name := appMeshFirstNonEmpty(appMeshString(payload, "routeName", ""), "stackyard-route")
		rt := s.ensureRouteLocked(meshName, virtualRouterName, name, now)
		appMeshMergeSpec(rt, payload)
		appMeshTouchMetadata(rt, now)
		return map[string]any{"route": appMeshCloneMap(rt)}

	case "DeleteRoute":
		meshName := appMeshFirstNonEmpty(appMeshString(pathParams, "meshName", ""), "apps")
		virtualRouterName := appMeshFirstNonEmpty(appMeshString(pathParams, "virtualRouterName", ""), "stackyard-virtual-router")
		name := appMeshFirstNonEmpty(appMeshString(pathParams, "routeName", ""), appMeshString(payload, "routeName", ""), "stackyard-route")
		rt := s.ensureRouteLocked(meshName, virtualRouterName, name, now)
		appMeshSetStatus(rt, "DELETED")
		appMeshTouchMetadata(rt, now)
		delete(s.ensureRouteMapLocked(meshName, virtualRouterName), name)
		return map[string]any{"route": appMeshCloneMap(rt)}

	case "DescribeRoute":
		meshName := appMeshFirstNonEmpty(appMeshString(pathParams, "meshName", ""), "apps")
		virtualRouterName := appMeshFirstNonEmpty(appMeshString(pathParams, "virtualRouterName", ""), "stackyard-virtual-router")
		name := appMeshFirstNonEmpty(appMeshString(pathParams, "routeName", ""), appMeshString(payload, "routeName", ""), "stackyard-route")
		rt := s.ensureRouteLocked(meshName, virtualRouterName, name, now)
		return map[string]any{"route": appMeshCloneMap(rt)}

	case "ListRoutes":
		meshName := appMeshFirstNonEmpty(appMeshString(pathParams, "meshName", ""), appMeshString(payload, "meshName", ""), "apps")
		virtualRouterName := appMeshFirstNonEmpty(appMeshString(pathParams, "virtualRouterName", ""), appMeshString(payload, "virtualRouterName", ""), "stackyard-virtual-router")
		return map[string]any{"routes": appMeshListNamed(s.ensureRouteMapLocked(meshName, virtualRouterName), "routeName"), "nextToken": ""}

	case "UpdateRoute":
		meshName := appMeshFirstNonEmpty(appMeshString(pathParams, "meshName", ""), "apps")
		virtualRouterName := appMeshFirstNonEmpty(appMeshString(pathParams, "virtualRouterName", ""), "stackyard-virtual-router")
		name := appMeshFirstNonEmpty(appMeshString(pathParams, "routeName", ""), appMeshString(payload, "routeName", ""), "stackyard-route")
		rt := s.ensureRouteLocked(meshName, virtualRouterName, name, now)
		appMeshMergeSpec(rt, payload)
		appMeshTouchMetadata(rt, now)
		return map[string]any{"route": appMeshCloneMap(rt)}

	case "CreateVirtualService":
		meshName := appMeshFirstNonEmpty(appMeshString(pathParams, "meshName", ""), appMeshString(payload, "meshName", ""), "apps")
		name := appMeshFirstNonEmpty(appMeshString(payload, "virtualServiceName", ""), "stackyard.local")
		vs := s.ensureVirtualServiceLocked(meshName, name, now)
		appMeshMergeSpec(vs, payload)
		appMeshTouchMetadata(vs, now)
		return map[string]any{"virtualService": appMeshCloneMap(vs)}

	case "DeleteVirtualService":
		meshName := appMeshFirstNonEmpty(appMeshString(pathParams, "meshName", ""), "apps")
		name := appMeshFirstNonEmpty(appMeshString(pathParams, "virtualServiceName", ""), appMeshString(payload, "virtualServiceName", ""), "stackyard.local")
		vs := s.ensureVirtualServiceLocked(meshName, name, now)
		appMeshSetStatus(vs, "DELETED")
		appMeshTouchMetadata(vs, now)
		delete(s.ensureVirtualServiceMapLocked(meshName), name)
		return map[string]any{"virtualService": appMeshCloneMap(vs)}

	case "DescribeVirtualService":
		meshName := appMeshFirstNonEmpty(appMeshString(pathParams, "meshName", ""), "apps")
		name := appMeshFirstNonEmpty(appMeshString(pathParams, "virtualServiceName", ""), appMeshString(payload, "virtualServiceName", ""), "stackyard.local")
		vs := s.ensureVirtualServiceLocked(meshName, name, now)
		return map[string]any{"virtualService": appMeshCloneMap(vs)}

	case "ListVirtualServices":
		meshName := appMeshFirstNonEmpty(appMeshString(pathParams, "meshName", ""), appMeshString(payload, "meshName", ""), "apps")
		return map[string]any{"virtualServices": appMeshListNamed(s.ensureVirtualServiceMapLocked(meshName), "virtualServiceName"), "nextToken": ""}

	case "UpdateVirtualService":
		meshName := appMeshFirstNonEmpty(appMeshString(pathParams, "meshName", ""), "apps")
		name := appMeshFirstNonEmpty(appMeshString(pathParams, "virtualServiceName", ""), appMeshString(payload, "virtualServiceName", ""), "stackyard.local")
		vs := s.ensureVirtualServiceLocked(meshName, name, now)
		appMeshMergeSpec(vs, payload)
		appMeshTouchMetadata(vs, now)
		return map[string]any{"virtualService": appMeshCloneMap(vs)}

	case "TagResource":
		resourceARN := appMeshFirstNonEmpty(appMeshString(payload, "resourceArn", ""), appMeshString(payload, "ResourceArn", ""), appMeshMeshARN("apps"))
		tags := appMeshTagsFromAny(payload["tags"])
		if len(tags) == 0 {
			tags = appMeshTagsFromAny(payload["Tags"])
		}
		if s.tags[resourceARN] == nil {
			s.tags[resourceARN] = map[string]string{}
		}
		for k, v := range tags {
			s.tags[resourceARN][k] = v
		}
		return map[string]any{}

	case "UntagResource":
		resourceARN := appMeshFirstNonEmpty(appMeshString(payload, "resourceArn", ""), appMeshString(payload, "ResourceArn", ""), appMeshMeshARN("apps"))
		keys := appMeshStringSlice(payload["tagKeys"])
		if len(keys) == 0 {
			keys = appMeshStringSlice(payload["TagKeys"])
		}
		for _, key := range keys {
			delete(s.tags[resourceARN], key)
		}
		return map[string]any{}

	case "ListTagsForResource":
		resourceARN := appMeshFirstNonEmpty(
			query.Get("resourceArn"),
			query.Get("ResourceArn"),
			appMeshString(payload, "resourceArn", ""),
			appMeshString(payload, "ResourceArn", ""),
			appMeshMeshARN("apps"),
		)
		return map[string]any{"tags": appMeshCloneStringMap(s.tags[resourceARN])}
	}

	return map[string]any{}
}

func (s *appMeshStore) syncPayloadWithQuery(payload map[string]any, query url.Values) {
	for key, values := range query {
		if len(values) == 0 {
			continue
		}
		if _, exists := payload[key]; exists {
			continue
		}
		if len(values) == 1 {
			payload[key] = values[0]
			continue
		}
		items := make([]any, 0, len(values))
		for _, v := range values {
			items = append(items, v)
		}
		payload[key] = items
	}
}

func (s *appMeshStore) ensureMeshLocked(meshName, now string) map[string]any {
	meshName = appMeshFirstNonEmpty(meshName, "apps")
	if existing, ok := s.meshes[meshName]; ok {
		return existing
	}
	mesh := s.newMesh(meshName, now)
	s.meshes[meshName] = mesh
	return mesh
}

func (s *appMeshStore) ensureVirtualGatewayLocked(meshName, name, now string) map[string]any {
	s.ensureMeshLocked(meshName, now)
	name = appMeshFirstNonEmpty(name, "stackyard-virtual-gateway")
	m := s.ensureVirtualGatewayMapLocked(meshName)
	if existing, ok := m[name]; ok {
		return existing
	}
	item := s.newVirtualGateway(meshName, name, now)
	m[name] = item
	return item
}

func (s *appMeshStore) ensureGatewayRouteLocked(meshName, virtualGatewayName, name, now string) map[string]any {
	s.ensureVirtualGatewayLocked(meshName, virtualGatewayName, now)
	name = appMeshFirstNonEmpty(name, "stackyard-gateway-route")
	m := s.ensureGatewayRouteMapLocked(meshName, virtualGatewayName)
	if existing, ok := m[name]; ok {
		return existing
	}
	item := s.newGatewayRoute(meshName, virtualGatewayName, name, now)
	m[name] = item
	return item
}

func (s *appMeshStore) ensureVirtualNodeLocked(meshName, name, now string) map[string]any {
	s.ensureMeshLocked(meshName, now)
	name = appMeshFirstNonEmpty(name, "stackyard-virtual-node")
	m := s.ensureVirtualNodeMapLocked(meshName)
	if existing, ok := m[name]; ok {
		return existing
	}
	item := s.newVirtualNode(meshName, name, now)
	m[name] = item
	return item
}

func (s *appMeshStore) ensureVirtualRouterLocked(meshName, name, now string) map[string]any {
	s.ensureMeshLocked(meshName, now)
	name = appMeshFirstNonEmpty(name, "stackyard-virtual-router")
	m := s.ensureVirtualRouterMapLocked(meshName)
	if existing, ok := m[name]; ok {
		return existing
	}
	item := s.newVirtualRouter(meshName, name, now)
	m[name] = item
	return item
}

func (s *appMeshStore) ensureRouteLocked(meshName, virtualRouterName, name, now string) map[string]any {
	s.ensureVirtualRouterLocked(meshName, virtualRouterName, now)
	name = appMeshFirstNonEmpty(name, "stackyard-route")
	m := s.ensureRouteMapLocked(meshName, virtualRouterName)
	if existing, ok := m[name]; ok {
		return existing
	}
	item := s.newRoute(meshName, virtualRouterName, name, now)
	m[name] = item
	return item
}

func (s *appMeshStore) ensureVirtualServiceLocked(meshName, name, now string) map[string]any {
	s.ensureMeshLocked(meshName, now)
	name = appMeshFirstNonEmpty(name, "stackyard.local")
	m := s.ensureVirtualServiceMapLocked(meshName)
	if existing, ok := m[name]; ok {
		return existing
	}
	item := s.newVirtualService(meshName, name, now)
	m[name] = item
	return item
}

func (s *appMeshStore) ensureVirtualGatewayMapLocked(meshName string) map[string]map[string]any {
	if s.virtualGateways[meshName] == nil {
		s.virtualGateways[meshName] = map[string]map[string]any{}
	}
	return s.virtualGateways[meshName]
}

func (s *appMeshStore) ensureGatewayMeshMapLocked(meshName string) map[string]map[string]map[string]any {
	if s.gatewayRoutes[meshName] == nil {
		s.gatewayRoutes[meshName] = map[string]map[string]map[string]any{}
	}
	return s.gatewayRoutes[meshName]
}

func (s *appMeshStore) ensureGatewayRouteMapLocked(meshName, virtualGatewayName string) map[string]map[string]any {
	meshMap := s.ensureGatewayMeshMapLocked(meshName)
	if meshMap[virtualGatewayName] == nil {
		meshMap[virtualGatewayName] = map[string]map[string]any{}
	}
	return meshMap[virtualGatewayName]
}

func (s *appMeshStore) ensureVirtualNodeMapLocked(meshName string) map[string]map[string]any {
	if s.virtualNodes[meshName] == nil {
		s.virtualNodes[meshName] = map[string]map[string]any{}
	}
	return s.virtualNodes[meshName]
}

func (s *appMeshStore) ensureVirtualRouterMapLocked(meshName string) map[string]map[string]any {
	if s.virtualRouters[meshName] == nil {
		s.virtualRouters[meshName] = map[string]map[string]any{}
	}
	return s.virtualRouters[meshName]
}

func (s *appMeshStore) ensureRouteRouterMapLocked(meshName string) map[string]map[string]map[string]any {
	if s.routes[meshName] == nil {
		s.routes[meshName] = map[string]map[string]map[string]any{}
	}
	return s.routes[meshName]
}

func (s *appMeshStore) ensureRouteMapLocked(meshName, virtualRouterName string) map[string]map[string]any {
	meshMap := s.ensureRouteRouterMapLocked(meshName)
	if meshMap[virtualRouterName] == nil {
		meshMap[virtualRouterName] = map[string]map[string]any{}
	}
	return meshMap[virtualRouterName]
}

func (s *appMeshStore) ensureVirtualServiceMapLocked(meshName string) map[string]map[string]any {
	if s.virtualServices[meshName] == nil {
		s.virtualServices[meshName] = map[string]map[string]any{}
	}
	return s.virtualServices[meshName]
}

func (s *appMeshStore) newMesh(meshName, now string) map[string]any {
	return map[string]any{
		"meshName":      meshName,
		"meshOwner":     appMeshDefaultAccount,
		"resourceOwner": appMeshDefaultAccount,
		"metadata":      appMeshMetadata(appMeshMeshARN(meshName), now),
		"spec": map[string]any{
			"egressFilter": map[string]any{"type": "ALLOW_ALL"},
		},
		"status": map[string]any{"status": "ACTIVE"},
	}
}

func (s *appMeshStore) newVirtualGateway(meshName, name, now string) map[string]any {
	return map[string]any{
		"meshName":           meshName,
		"meshOwner":          appMeshDefaultAccount,
		"virtualGatewayName": name,
		"resourceOwner":      appMeshDefaultAccount,
		"metadata":           appMeshMetadata(appMeshVirtualGatewayARN(meshName, name), now),
		"spec": map[string]any{
			"listeners": []any{map[string]any{"portMapping": map[string]any{"port": 8080, "protocol": "http"}}},
		},
		"status": map[string]any{"status": "ACTIVE"},
	}
}

func (s *appMeshStore) newGatewayRoute(meshName, virtualGatewayName, name, now string) map[string]any {
	return map[string]any{
		"meshName":           meshName,
		"meshOwner":          appMeshDefaultAccount,
		"virtualGatewayName": virtualGatewayName,
		"gatewayRouteName":   name,
		"resourceOwner":      appMeshDefaultAccount,
		"metadata":           appMeshMetadata(appMeshGatewayRouteARN(meshName, virtualGatewayName, name), now),
		"spec": map[string]any{
			"httpRoute": map[string]any{"match": map[string]any{"prefix": "/"}},
		},
		"status": map[string]any{"status": "ACTIVE"},
	}
}

func (s *appMeshStore) newVirtualNode(meshName, name, now string) map[string]any {
	return map[string]any{
		"meshName":        meshName,
		"meshOwner":       appMeshDefaultAccount,
		"virtualNodeName": name,
		"resourceOwner":   appMeshDefaultAccount,
		"metadata":        appMeshMetadata(appMeshVirtualNodeARN(meshName, name), now),
		"spec": map[string]any{
			"listeners": []any{map[string]any{"portMapping": map[string]any{"port": 8080, "protocol": "http"}}},
		},
		"status": map[string]any{"status": "ACTIVE"},
	}
}

func (s *appMeshStore) newVirtualRouter(meshName, name, now string) map[string]any {
	return map[string]any{
		"meshName":          meshName,
		"meshOwner":         appMeshDefaultAccount,
		"virtualRouterName": name,
		"resourceOwner":     appMeshDefaultAccount,
		"metadata":          appMeshMetadata(appMeshVirtualRouterARN(meshName, name), now),
		"spec": map[string]any{
			"listeners": []any{map[string]any{"portMapping": map[string]any{"port": 8080, "protocol": "http"}}},
		},
		"status": map[string]any{"status": "ACTIVE"},
	}
}

func (s *appMeshStore) newRoute(meshName, virtualRouterName, name, now string) map[string]any {
	return map[string]any{
		"meshName":          meshName,
		"meshOwner":         appMeshDefaultAccount,
		"virtualRouterName": virtualRouterName,
		"routeName":         name,
		"resourceOwner":     appMeshDefaultAccount,
		"metadata":          appMeshMetadata(appMeshRouteARN(meshName, virtualRouterName, name), now),
		"spec": map[string]any{
			"httpRoute": map[string]any{"match": map[string]any{"prefix": "/"}},
		},
		"status": map[string]any{"status": "ACTIVE"},
	}
}

func (s *appMeshStore) newVirtualService(meshName, name, now string) map[string]any {
	return map[string]any{
		"meshName":           meshName,
		"meshOwner":          appMeshDefaultAccount,
		"virtualServiceName": name,
		"resourceOwner":      appMeshDefaultAccount,
		"metadata":           appMeshMetadata(appMeshVirtualServiceARN(meshName, name), now),
		"spec": map[string]any{
			"provider": map[string]any{"virtualRouter": map[string]any{"virtualRouterName": "stackyard-router"}},
		},
		"status": map[string]any{"status": "ACTIVE"},
	}
}

func appMeshListMeshes(items map[string]map[string]any) []any {
	keys := make([]string, 0, len(items))
	for key := range items {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]any, 0, len(keys))
	for _, key := range keys {
		out = append(out, appMeshCloneMap(items[key]))
	}
	return out
}

func appMeshListNamed(items map[string]map[string]any, nameKey string) []any {
	keys := make([]string, 0, len(items))
	for key := range items {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]any, 0, len(keys))
	for _, key := range keys {
		cloned := appMeshCloneMap(items[key])
		if _, exists := cloned[nameKey]; !exists {
			cloned[nameKey] = key
		}
		out = append(out, cloned)
	}
	return out
}

func appMeshMergeSpec(target map[string]any, payload map[string]any) {
	specRaw, ok := payload["spec"]
	if !ok {
		return
	}
	spec, ok := specRaw.(map[string]any)
	if !ok {
		return
	}
	target["spec"] = appMeshCloneMap(spec)
}

func appMeshSetStatus(target map[string]any, status string) {
	target["status"] = map[string]any{"status": status}
}

func appMeshTouchMetadata(target map[string]any, now string) {
	metadata, ok := target["metadata"].(map[string]any)
	if !ok || metadata == nil {
		metadata = map[string]any{}
		target["metadata"] = metadata
	}
	metadata["lastUpdatedAt"] = now
	if _, exists := metadata["createdAt"]; !exists {
		metadata["createdAt"] = now
	}
	if _, exists := metadata["version"]; !exists {
		metadata["version"] = int64(1)
	}
	if _, exists := metadata["uid"]; !exists {
		metadata["uid"] = fmt.Sprintf("uid-%d", time.Now().UnixNano())
	}
}

func appMeshMetadata(arn, now string) map[string]any {
	return map[string]any{
		"arn":           arn,
		"createdAt":     now,
		"lastUpdatedAt": now,
		"uid":           fmt.Sprintf("uid-%d", time.Now().UnixNano()),
		"version":       int64(1),
	}
}

const (
	appMeshDefaultRegion  = "us-east-1"
	appMeshDefaultAccount = "123456789012"
)

func appMeshMeshARN(meshName string) string {
	return fmt.Sprintf("arn:aws:appmesh:%s:%s:mesh/%s", appMeshDefaultRegion, appMeshDefaultAccount, meshName)
}

func appMeshVirtualGatewayARN(meshName, name string) string {
	return fmt.Sprintf("%s/virtualGateway/%s", appMeshMeshARN(meshName), name)
}

func appMeshGatewayRouteARN(meshName, virtualGatewayName, name string) string {
	return fmt.Sprintf("%s/virtualGateway/%s/gatewayRoute/%s", appMeshMeshARN(meshName), virtualGatewayName, name)
}

func appMeshVirtualNodeARN(meshName, name string) string {
	return fmt.Sprintf("%s/virtualNode/%s", appMeshMeshARN(meshName), name)
}

func appMeshVirtualRouterARN(meshName, name string) string {
	return fmt.Sprintf("%s/virtualRouter/%s", appMeshMeshARN(meshName), name)
}

func appMeshRouteARN(meshName, virtualRouterName, name string) string {
	return fmt.Sprintf("%s/virtualRouter/%s/route/%s", appMeshMeshARN(meshName), virtualRouterName, name)
}

func appMeshVirtualServiceARN(meshName, name string) string {
	return fmt.Sprintf("%s/virtualService/%s", appMeshMeshARN(meshName), name)
}

func appMeshString(obj any, key, def string) string {
	switch typed := obj.(type) {
	case map[string]string:
		s := strings.TrimSpace(typed[key])
		if s == "" {
			return def
		}
		return s
	case map[string]any:
		raw, ok := typed[key]
		if !ok || raw == nil {
			return def
		}
		s, ok := raw.(string)
		if !ok {
			return def
		}
		s = strings.TrimSpace(s)
		if s == "" {
			return def
		}
		return s
	default:
		return def
	}
}

func appMeshFirstNonEmpty(values ...string) string {
	for _, value := range values {
		v := strings.TrimSpace(value)
		if v != "" {
			return v
		}
	}
	return ""
}

func appMeshStringSlice(raw any) []string {
	if raw == nil {
		return nil
	}
	if items, ok := raw.([]string); ok {
		out := make([]string, 0, len(items))
		for _, item := range items {
			item = strings.TrimSpace(item)
			if item != "" {
				out = append(out, item)
			}
		}
		return out
	}
	list, ok := raw.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(list))
	for _, item := range list {
		s, ok := item.(string)
		if !ok {
			continue
		}
		s = strings.TrimSpace(s)
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}

func appMeshTagsFromAny(raw any) map[string]string {
	if raw == nil {
		return map[string]string{}
	}
	if tags, ok := raw.(map[string]string); ok {
		return appMeshCloneStringMap(tags)
	}
	if tags, ok := raw.(map[string]any); ok {
		out := map[string]string{}
		for key, value := range tags {
			k := strings.TrimSpace(key)
			if k == "" {
				continue
			}
			out[k] = fmt.Sprintf("%v", value)
		}
		return out
	}
	if list, ok := raw.([]any); ok {
		out := map[string]string{}
		for _, item := range list {
			tag, ok := item.(map[string]any)
			if !ok {
				continue
			}
			key := appMeshFirstNonEmpty(appMeshString(tag, "key", ""), appMeshString(tag, "Key", ""))
			value := appMeshFirstNonEmpty(appMeshString(tag, "value", ""), appMeshString(tag, "Value", ""))
			if key != "" {
				out[key] = value
			}
		}
		return out
	}
	return map[string]string{}
}

func appMeshCloneMap(in map[string]any) map[string]any {
	if in == nil {
		return nil
	}
	out := make(map[string]any, len(in))
	for key, value := range in {
		switch typed := value.(type) {
		case map[string]any:
			out[key] = appMeshCloneMap(typed)
		case []any:
			out[key] = appMeshCloneSlice(typed)
		case map[string]string:
			out[key] = appMeshCloneStringMap(typed)
		default:
			out[key] = typed
		}
	}
	return out
}

func appMeshCloneSlice(in []any) []any {
	out := make([]any, 0, len(in))
	for _, item := range in {
		switch typed := item.(type) {
		case map[string]any:
			out = append(out, appMeshCloneMap(typed))
		case []any:
			out = append(out, appMeshCloneSlice(typed))
		case map[string]string:
			out = append(out, appMeshCloneStringMap(typed))
		default:
			out = append(out, typed)
		}
	}
	return out
}

func appMeshCloneStringMap(in map[string]string) map[string]string {
	out := map[string]string{}
	for key, value := range in {
		out[key] = value
	}
	return out
}
