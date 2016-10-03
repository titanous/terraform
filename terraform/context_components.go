package terraform

// contextComponentFactory is the interface that Context uses
// to initialize various components such as providers and provisioners.
// This factory gets more information than the raw maps using to initialize
// a Context. This information is used for debugging.
type contextComponentFactory interface {
	// ResourceProvider creates a new ResourceProvider with the given
	// type. The "uid" is a unique identifier for this provider being
	// initialized that can be used for internal tracking.
	ResourceProvider(typ, uid string) (ResourceProvider, error)

	// ResourceProvisioner creates a new ResourceProvisioner with the
	// given type. The "uid" is a unique identifier for this provisioner
	// being initialized that can be used for internal tracking.
	ResourceProvisioner(typ, uid string) (ResourceProvisioner, error)
}
