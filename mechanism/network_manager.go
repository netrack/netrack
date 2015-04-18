package mech

type NetworkMechanismMap map[string]NetworkMechanism

func (m NetworkMechanismMap) Get(s string) (Mechanism, bool) {
	return m[s]
}

func (m NetworkMechanismMap) Set(s string, mechanism Mechanism) {
	m[s] = mechanism
}

func (m NetworkMechanismMap) Iter(fn func(string, Mechanism) bool) {
	for s, mechansim := range m {
		fn(s, mechanism)
	}
}

type NetworkMechanismManager struct {
	MechanismManager
}
