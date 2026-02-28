package server

type identityStoreDataType struct {
	Name string
}

// AWS Identity Store data types sourced from:
// https://docs.aws.amazon.com/singlesignon/latest/IdentityStoreAPIReference/API_Types.html
var identityStoreDataTypes = []identityStoreDataType{
	{Name: "Address"},
	{Name: "AlternateIdentifier"},
	{Name: "AttributeOperation"},
	{Name: "Email"},
	{Name: "ExternalId"},
	{Name: "Filter"},
	{Name: "Group"},
	{Name: "GroupMembership"},
	{Name: "GroupMembershipExistenceResult"},
	{Name: "MemberId"},
	{Name: "Name"},
	{Name: "PhoneNumber"},
	{Name: "Photo"},
	{Name: "Role"},
	{Name: "UniqueAttribute"},
	{Name: "UpdateUser"},
	{Name: "User"},
}

var identityStoreDataTypeByName = func() map[string]identityStoreDataType {
	out := make(map[string]identityStoreDataType, len(identityStoreDataTypes))
	for _, dt := range identityStoreDataTypes {
		out[dt.Name] = dt
	}
	return out
}()
