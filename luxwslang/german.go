package luxwslang

import (
	"regexp"
)

// German language terminology.
var German = &Terminology{
	ID:   "de",
	Name: "Deutsch",

	timestampFormat: "02.01.06 15:04:05",

	NavInformation:  "Informationen",
	NavTemperatures: "Temperaturen",
	NavElapsedTimes: "Ablaufzeiten",
	NavInputs:       "Eingänge",
	NavOutputs:      "Ausgänge",
	NavHeatQuantity: "Wärmemenge",
	NavErrorMemory:  "Fehlerspeicher",
	NavSwitchOffs:   "Abschaltungen",

	NavOpHours:      "Betriebsstunden",
	HoursImpulsesRe: regexp.MustCompile(`^Impulse\s`),

	NavSystemStatus:       "Anlagenstatus",
	StatusType:            "Wärmepumpen Typ",
	StatusSoftwareVersion: "Softwarestand",
	StatusOperationMode:   "Betriebszustand",
	StatusPowerOutput:     "Leistung Ist",

	BoolFalse: "Aus",
	BoolTrue:  "Ein",
}
