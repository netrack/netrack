package mech

type LinkMechanismMap map[string]LinkMechanism

func (m LinkMechanismMap) Get(s string) (Mechanism, bool) {
	return m[s]
}

func (m LinkMechanismMap) Set(s string, mechanism Mechanism) {
	m[s] = mechanism
}

func (m LinkMechanismMap) Iter(fn func(string, Mechanism) bool) {
	for s, mechansim := range m {
		fn(s, mechanism)
	}
}

type LinkMechanismManager struct {
	MechanismManager
}
