package rds

import (
	"strings"
	"time"
)

// DescribeDBClusterByRef returns the cluster named by identifier or ARN.
func (s *Service) DescribeDBClusterByRef(ref string) (DBCluster, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cluster, err := s.clusterByRefLocked(ref)
	if err != nil {
		return DBCluster{}, err
	}
	return cloneDBCluster(cluster), nil
}

// SetDBClusterHTTPEndpoint turns the RDS Data API endpoint on or off for the
// cluster named by identifier or ARN (EnableHttpEndpoint/DisableHttpEndpoint).
func (s *Service) SetDBClusterHTTPEndpoint(ref string, enabled bool) (DBCluster, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cluster, err := s.clusterByRefLocked(ref)
	if err != nil {
		return DBCluster{}, err
	}
	cluster.HTTPEndpointEnabled = enabled
	cluster.UpdatedAt = time.Now().UTC()
	return cloneDBCluster(cluster), nil
}

func (s *Service) clusterByRefLocked(ref string) (*DBCluster, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return nil, ErrInvalidParameter
	}
	if cluster, ok := s.clusters[ref]; ok {
		return cluster, nil
	}
	for _, cluster := range s.clusters {
		if cluster.ARN == ref {
			return cluster, nil
		}
	}
	return nil, ErrNotFound
}
