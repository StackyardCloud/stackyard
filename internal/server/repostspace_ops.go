package server

// AWS re:Post Private operations sourced from:
// https://docs.aws.amazon.com/repostprivate/latest/APIReference/API_Operations.html
var repostSpaceOperations = []rpsOperation{
	{Name: "BatchAddChannelRoleToAccessors", Method: "POST", Pattern: "/spaces/{spaceId}/channels/{channelId}/roles"},
	{Name: "BatchAddRole", Method: "POST", Pattern: "/spaces/{spaceId}/roles"},
	{Name: "BatchRemoveChannelRoleFromAccessors", Method: "PATCH", Pattern: "/spaces/{spaceId}/channels/{channelId}/roles"},
	{Name: "BatchRemoveRole", Method: "PATCH", Pattern: "/spaces/{spaceId}/roles"},
	{Name: "CreateChannel", Method: "POST", Pattern: "/spaces/{spaceId}/channels"},
	{Name: "CreateSpace", Method: "POST", Pattern: "/spaces"},
	{Name: "DeleteSpace", Method: "DELETE", Pattern: "/spaces/{spaceId}"},
	{Name: "DeregisterAdmin", Method: "DELETE", Pattern: "/spaces/{spaceId}/admins/{adminId}"},
	{Name: "GetChannel", Method: "GET", Pattern: "/spaces/{spaceId}/channels/{channelId}"},
	{Name: "GetSpace", Method: "GET", Pattern: "/spaces/{spaceId}"},
	{Name: "ListChannels", Method: "GET", Pattern: "/spaces/{spaceId}/channels"},
	{Name: "ListSpaces", Method: "GET", Pattern: "/spaces"},
	{Name: "ListTagsForResource", Method: "GET", Pattern: "/tags/{resourceArn}"},
	{Name: "RegisterAdmin", Method: "POST", Pattern: "/spaces/{spaceId}/admins/{adminId}"},
	{Name: "SendInvites", Method: "POST", Pattern: "/spaces/{spaceId}/invite"},
	{Name: "TagResource", Method: "POST", Pattern: "/tags/{resourceArn}"},
	{Name: "UntagResource", Method: "DELETE", Pattern: "/tags/{resourceArn}"},
	{Name: "UpdateChannel", Method: "PUT", Pattern: "/spaces/{spaceId}/channels/{channelId}"},
	{Name: "UpdateSpace", Method: "PUT", Pattern: "/spaces/{spaceId}"},
}
