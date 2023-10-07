package cluster

// MembershipChanged is an engine event that indicates a change in membership to
// the cluster.
type MembershipChanged struct {
	Registered, Deregistered []Node
}
