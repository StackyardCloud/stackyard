package server

import (
	"net/http"
	"net/netip"
	"strconv"
	"strings"
)

// ExpressRoute routes outside the shared ARM handler's resource model:
// circuit and peering statistics (GET .../stats) and the ARP table, route
// table and route table summary of a circuit or cross-connection peering
// (POST .../peerings/{peering}/{table}/{devicePath}). Tables are synthesized
// from the peering's configuration; real ones answer 202 and finish later,
// these finish at once.

// xrServeTables handles the routes above, returning false for anything else.
func (s *Server) xrServeTables(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !strings.HasPrefix(path, "/azure/subscriptions/") {
		return false
	}
	seg := splitPathSegments(strings.TrimPrefix(path, "/azure/"))
	// subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.Network/{type}/{name}/...
	if len(seg) < 9 || !strings.EqualFold(seg[2], "resourceGroups") || !strings.EqualFold(seg[4], "providers") || !strings.EqualFold(seg[5], "Microsoft.Network") {
		return false
	}
	parentType := seg[6]
	if !strings.EqualFold(parentType, "expressRouteCircuits") && !strings.EqualFold(parentType, "expressRouteCrossConnections") {
		return false
	}
	parentID := "/subscriptions/" + seg[1] + "/resourceGroups/" + seg[3] + "/providers/Microsoft.Network/" + parentType + "/" + seg[7]
	rest := seg[8:]
	circuit := strings.EqualFold(parentType, "expressRouteCircuits")
	switch {
	case circuit && len(rest) == 1 && strings.EqualFold(rest[0], "stats"):
		s.xrServeStats(w, r, parentID)
	case circuit && len(rest) == 3 && strings.EqualFold(rest[0], "peerings") && strings.EqualFold(rest[2], "stats"):
		s.xrServeStats(w, r, parentID+"/peerings/"+rest[1])
	case len(rest) == 4 && strings.EqualFold(rest[0], "peerings"):
		table := strings.ToLower(rest[2])
		if table != "arptables" && table != "routetables" && table != "routetablessummary" {
			return false
		}
		s.serveAzureARMPost(w, r, func(map[string]any) (int, any) {
			return s.xrTable(parentID+"/peerings/"+rest[1], table, rest[3])
		})
	default:
		return false
	}
	return true
}

func (s *Server) xrServeStats(w http.ResponseWriter, r *http.Request, id string) {
	switch {
	case r.Method != http.MethodGet:
		azureARMError(w, http.StatusMethodNotAllowed, "MethodNotAllowed", "method not allowed")
	case hasAzureInvalidAPIVersion(r):
		azureARMError(w, http.StatusBadRequest, "InvalidApiVersionParameter", "invalid api-version query parameter")
	case s.azureARMLookup(id) == nil:
		azureARMNotFound(w, "ResourceNotFound", id, "")
	default:
		// No traffic flows through the emulator.
		respondJSON(w, http.StatusOK, map[string]any{"primarybytesIn": 0, "primarybytesOut": 0, "secondarybytesIn": 0, "secondarybytesOut": 0})
	}
}

// xrTable returns a peering's ARP table, route table or route table summary
// on the primary or secondary device.
func (s *Server) xrTable(peeringID, table, device string) (int, any) {
	if device != "primary" && device != "secondary" {
		return http.StatusBadRequest, azureARMErrorBody("InvalidRequest", "The device path must be primary or secondary.")
	}
	peering := s.azureARMLookup(peeringID)
	if peering == nil {
		return http.StatusNotFound, azureARMErrorBody("ResourceNotFound", "The peering "+peeringID+" was not found.")
	}
	prefix, err := netip.ParsePrefix(azureARMString(peering.Properties[device+"PeerAddressPrefix"]))
	if err != nil {
		return http.StatusOK, map[string]any{"value": []any{}}
	}
	// In a /30 the customer router takes the first host address and Azure the second.
	customer := prefix.Masked().Addr().Next()
	azure := customer.Next()
	asn := peering.Properties["peerASN"]
	asnText := ""
	if n, ok := asn.(float64); ok {
		asnText = strconv.FormatInt(int64(n), 10)
	}
	switch table {
	case "arptables":
		return http.StatusOK, map[string]any{"value": []any{
			map[string]any{"age": 0, "interface": "On-Prem", "ipAddress": customer.String(), "macAddress": "00:00:5e:00:53:01"},
			map[string]any{"age": 0, "interface": "Microsoft", "ipAddress": azure.String(), "macAddress": "00:00:5e:00:53:02"},
		}}
	case "routetables":
		var routes []any
		for _, advertised := range azureARMSlice(azureARMObject(peering.Properties["microsoftPeeringConfig"])["advertisedPublicPrefixes"]) {
			routes = append(routes, map[string]any{"network": advertised, "nextHop": customer.String(), "locPrf": "", "weight": 0, "path": asnText})
		}
		if routes == nil {
			routes = []any{}
		}
		return http.StatusOK, map[string]any{"value": routes}
	default:
		return http.StatusOK, map[string]any{"value": []any{map[string]any{
			"neighbor": customer.String(), "v": 4, "as": asn, "upDown": "00:00:00", "statePfxRcd": "0",
		}}}
	}
}
