package luxwslang

import "regexp"

// Dutch language terminology.
var Dutch = &Terminology{
	ID:   "nl",
	Name: "Nederlands",

	timestampFormat: "02.01.06 15:04:05",

	NavInformation:  "Informatie",
	NavTemperatures: "Temperaturen",
	NavElapsedTimes: "Aflooptijden",
	NavInputs:       "Ingangen",
	NavOutputs:      "Uitgangen",
	NavHeatQuantity: "Energie",
	NavErrorMemory:  "Storingsbuffer",
	NavSwitchOffs:   "Afschakelingen",

	NavOpHours:      "Bedrijfsuren",
	HoursImpulsesRe: regexp.MustCompile(`^impulse\s`),

	NavSystemStatus:       "Installatiestatus",
	StatusType:            "Warmtepomp Type",
	StatusSoftwareVersion: "Softwareversie",
	StatusOperationMode:   "Bedrijfstoestand",
	StatusPowerOutput:     "Vermogen",

	BoolFalse: "Uit",
	BoolTrue:  "Aan",
}
