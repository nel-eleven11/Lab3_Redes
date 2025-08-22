package main

type TopologyDB struct {
	lsaDatabase map[string]*LSAData // router -> LSA
}

func NewTopologyDB() *TopologyDB {
	return &TopologyDB{
		lsaDatabase: make(map[string]*LSAData),
	}
}

func (tdb *TopologyDB) UpdateLSA(lsa *LSAData) bool {
	existing, exists := tdb.lsaDatabase[lsa.OriginRouter]

	// Check if this is a newer LSA
	if !exists || lsa.SequenceNum > existing.SequenceNum {
		tdb.lsaDatabase[lsa.OriginRouter] = lsa
		return true // Updated
	}
	return false // Not updated
}

func (tdb *TopologyDB) GetLSA(router string) (*LSAData, bool) {
	lsa, exists := tdb.lsaDatabase[router]
	return lsa, exists
}

func (tdb *TopologyDB) GetAllLSAs() map[string]*LSAData {
	return tdb.lsaDatabase
}

func (tdb *TopologyDB) RemoveLSA(router string) {
	delete(tdb.lsaDatabase, router)
}
