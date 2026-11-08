package energyquerytype

// A EnergyQueryType contains information about a group of energyqueries with the same functionality.
type EnergyQueryType struct {
	ID                 uint   `json:"id"`
	EnergyQueryVariety string `json:"name"`
	Formula            string `json:"formula"`
	// Rounding rules (JSON, see RoundingRule) the app applies before uploading results.
	Rounding string `json:"rounding,omitempty"`
}

// Create a new EnergyQueryType.
func MakeEnergyQueryType(energyQueryVariety string, formula string, rounding string) EnergyQueryType {
	return EnergyQueryType{
		EnergyQueryVariety: energyQueryVariety,
		Formula:            formula,
		Rounding:           rounding,
	}
}
