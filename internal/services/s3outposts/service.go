package s3outposts

import (
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"
)

var (
	ErrEndpointNotFound = errors.New("endpoint not found")
)

type Endpoint struct {
	AccessType            string
	CidrBlock             string
	CreationTime          int64
	CustomerOwnedIpv4Pool string
	EndpointArn           string
	EndpointID            string
	NetworkInterfaceID    string
	NetworkType           string
	OutpostsID            string
	SecurityGroupID       string
	Status                string
	SubnetID              string
	VpcID                 string
}

type Outpost struct {
	CapacityInBytes int64
	OutpostArn      string
	OutpostID       string
	OwnerID         string
	S3OutpostArn    string
}

type Service struct {
	mu          sync.Mutex
	endpoints   map[string]Endpoint
	tombstones  map[string]bool
	idempotency map[string]string
	outposts    []Outpost
	nextID      int
}

func NewService() *Service {
	s := &Service{
		endpoints:   map[string]Endpoint{},
		tombstones:  map[string]bool{},
		idempotency: map[string]string{},
		outposts: []Outpost{
			{
				CapacityInBytes: 1099511627776,
				OutpostArn:      "arn:aws:outposts:us-east-1:123456789012:outpost/op-0123456789abcdef0",
				OutpostID:       "op-0123456789abcdef0",
				OwnerID:         "123456789012",
				S3OutpostArn:    "arn:aws:s3-outposts:us-east-1:123456789012:outpost/op-0123456789abcdef0",
			},
			{
				CapacityInBytes: 1099511627776,
				OutpostArn:      "arn:aws:outposts:us-east-1:123456789012:outpost/op-11111111111111111",
				OutpostID:       "op-11111111111111111",
				OwnerID:         "123456789012",
				S3OutpostArn:    "arn:aws:s3-outposts:us-east-1:123456789012:outpost/op-11111111111111111",
			},
		},
		nextID: 2,
	}
	seed := Endpoint{
		AccessType:            "Private",
		CidrBlock:             "10.0.0.0/24",
		CreationTime:          1700000000,
		CustomerOwnedIpv4Pool: "coip-pool-12345678",
		EndpointArn:           "arn:aws:s3-outposts:us-east-1:123456789012:outpost/op-0123456789abcdef0/endpoint/1234567890123456789",
		EndpointID:            "1234567890123456789",
		NetworkInterfaceID:    "eni-0123456789abcdef0",
		NetworkType:           "IPV4",
		OutpostsID:            "op-0123456789abcdef0",
		SecurityGroupID:       "sg-0123456789abcdef0",
		Status:                "Available",
		SubnetID:              "subnet-0123456789abcdef0",
		VpcID:                 "vpc-0123456789abcdef0",
	}
	s.endpoints[seed.EndpointID] = seed
	return s
}

func (s *Service) CreateEndpoint(outpostID, subnetID, securityGroupID, accessType, networkType, customerOwnedIpv4Pool, clientToken string) Endpoint {
	s.mu.Lock()
	defer s.mu.Unlock()
	if clientToken != "" {
		if existingID, ok := s.idempotency[clientToken]; ok {
			if ep, ok := s.endpoints[existingID]; ok {
				return ep
			}
		}
	}
	id := s.nextEndpointID()
	ep := Endpoint{
		AccessType:            accessType,
		CreationTime:          time.Now().UTC().Unix(),
		CustomerOwnedIpv4Pool: customerOwnedIpv4Pool,
		EndpointArn:           "arn:aws:s3-outposts:us-east-1:123456789012:outpost/" + outpostID + "/endpoint/" + id,
		EndpointID:            id,
		NetworkType:           networkType,
		OutpostsID:            outpostID,
		SecurityGroupID:       securityGroupID,
		Status:                "Available",
		SubnetID:              subnetID,
	}
	s.endpoints[ep.EndpointID] = ep
	if clientToken != "" {
		s.idempotency[clientToken] = ep.EndpointID
	}
	return ep
}

func (s *Service) nextEndpointID() string {
	id := "123456789012345678" + fmt.Sprintf("%d", s.nextID)
	if s.nextID == 1 {
		id = "1234567890123456789"
	}
	s.nextID++
	return id
}

func (s *Service) DeleteEndpoint(endpointID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.endpoints[endpointID]; ok {
		delete(s.endpoints, endpointID)
		s.tombstones[endpointID] = true
		return nil
	}
	if s.tombstones[endpointID] {
		return nil
	}
	return ErrEndpointNotFound
}

func (s *Service) ListEndpoints() []Endpoint {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Endpoint, 0, len(s.endpoints))
	for _, ep := range s.endpoints {
		out = append(out, ep)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].EndpointID < out[j].EndpointID
	})
	return out
}

func (s *Service) ListSharedEndpoints(outpostID string) []Endpoint {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Endpoint, 0)
	for _, ep := range s.endpoints {
		if ep.OutpostsID == outpostID {
			out = append(out, ep)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].EndpointID < out[j].EndpointID
	})
	return out
}

func (s *Service) ListOutpostsWithS3() []Outpost {
	s.mu.Lock()
	defer s.mu.Unlock()
	withEndpoints := map[string]bool{}
	for _, ep := range s.endpoints {
		withEndpoints[ep.OutpostsID] = true
	}
	out := make([]Outpost, 0, len(s.outposts))
	for _, op := range s.outposts {
		if withEndpoints[op.OutpostID] {
			out = append(out, op)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].OutpostID < out[j].OutpostID
	})
	return out
}
