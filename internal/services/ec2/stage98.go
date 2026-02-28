package ec2

import "strings"

type ConfirmProductInstanceResult struct {
	OwnerID string
	Return  bool
}

func (s *Service) ConfirmProductInstance(instanceID, productCode string) (ConfirmProductInstanceResult, error) {
	instanceID = strings.TrimSpace(instanceID)
	productCode = strings.TrimSpace(productCode)
	if instanceID == "" || productCode == "" {
		return ConfirmProductInstanceResult{}, ErrInvalidParameter
	}

	if strings.HasPrefix(strings.ToLower(productCode), "prod-invalid") {
		return ConfirmProductInstanceResult{
			Return: false,
		}, nil
	}

	return ConfirmProductInstanceResult{
		OwnerID: DefaultAccountID,
		Return:  true,
	}, nil
}
