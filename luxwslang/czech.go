package luxwslang

import (
	"regexp"
)

// Czech language terminology.
var Czech = &Terminology{
	ID:   "cz",
	Name: "Česky",

	timestampFormat: "02.01.06 15:04:05",

	NavInformation:  "Informace",
	NavTemperatures: "Teploty",
	NavElapsedTimes: "Doby chodu",
	NavInputs:       "Vstupy",
	NavOutputs:      "Výstupy",
	NavHeatQuantity: "Teplo",
	NavErrorMemory:  "Chybová paměť",
	NavSwitchOffs:   "Odepnutí",

	NavOpHours:      "Provozní hodiny",
	HoursImpulsesRe: regexp.MustCompile(`^Počet startů\s`),

	NavSystemStatus:       "Status zařízení",
	StatusType:            "Typ TČ",
	StatusSoftwareVersion: "Softwarová verze",
	StatusOperationMode:   "Provozní stav",
	StatusPowerOutput:     "Výkon",

	BoolFalse: "Vypnuto",
	BoolTrue:  "Zapnuto",
}
