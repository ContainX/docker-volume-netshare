package drivers

type mount struct {
	name        string
	connections int
}

type mountManager struct {
	mounts map[string]*mount
}

func NewVolumeManager() *mountManager {
	return &mountManager{
		mounts: map[string]*mount{},
	}
}

func (m *mountManager) HasMount(dest string) bool {
	_, found := m.mounts[dest]
	return found
}

func (m *mountManager) IsActiveMount(dest string) bool {
	c, found := m.mounts[dest]
	return found && c.connections > 0
}

func (m *mountManager) Count(dest string) int {
	c, found := m.mounts[dest]
	if found {
		return c.connections
	}
	return 0
}

func (m *mountManager) Add(dest, name string) {
	c, found := m.mounts[dest]
	if found && c.connections > 0 {
		m.Increment(dest)
	} else {
		m.mounts[dest] = &mount{name: name, connections: 1}
	}
}

func (m *mountManager) Increment(dest string) int {
	c, found := m.mounts[dest]
	if found {
		if c.connections > 0 {
			c.connections++
		}
		return c.connections
	}
	return 0
}

func (m *mountManager) Decrement(dest string) int {
	c, found := m.mounts[dest]
	if found {
		c.connections--
	}
	return 0
}
