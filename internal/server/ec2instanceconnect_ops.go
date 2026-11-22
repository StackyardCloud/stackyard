package server

type ec2instanceconnectOperation struct{ Name string }

// AWS EC2 Instance Connect operations sourced from:
// https://docs.aws.amazon.com/ec2-instance-connect/latest/APIReference/API_Operations.html
var ec2instanceconnectOperations = []ec2instanceconnectOperation{
	{Name: "SendSSHPublicKey"},
	{Name: "SendSerialConsoleSSHPublicKey"},
}

var ec2instanceconnectOperationByName = func() map[string]ec2instanceconnectOperation {
	out := make(map[string]ec2instanceconnectOperation, len(ec2instanceconnectOperations))
	for _, op := range ec2instanceconnectOperations {
		out[op.Name] = op
	}
	return out
}()
