package luxwslang

import "regexp"

// Finnish language terminology.
var Finnish = &Terminology{
	ID:   "fi",
	Name: "Suomi",

	timestampFormat: "02.01.06 15:04:05",

	NavInformation:  "Informaatio",
	NavTemperatures: "Lämpötilat",
	NavElapsedTimes: "Käyntiajat",
	NavInputs:       "Tilat sisäänmeno",
	NavOutputs:      "Tilat ulostulo",
	NavHeatQuantity: "Kalorimetri",
	NavErrorMemory:  "Häiriöloki",
	NavSwitchOffs:   "Pysähtymistieto",

	NavOpHours:      "Käyttötunnit",
	HoursImpulsesRe: regexp.MustCompile(`^impulse\s`),

	NavSystemStatus:       "Laitetiedot",
	StatusType:            "Lämpöpumpun tyyppi",
	StatusSoftwareVersion: "Ohjelmaversio",
	StatusOperationMode:   "Toimintatila",
	StatusPowerOutput:     "Kapasiteetti",

	BoolFalse: "Pois",
	BoolTrue:  "On",
}
