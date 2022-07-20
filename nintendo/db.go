package nintendo

// Colors for the profile image's background
var profileColorsEnum = map[int]string{
	0: "pink",
	1: "green",
	2: "yellow",
	3: "purple",
	4: "blue",
	5: "sun-yellow",
}

var stageEnum = map[int]string{
	0:    "battera",    // The Reef
	1:    "fujitsubo",  // Musselforge Fitness
	2:    "gangaze",    // Starfish Mainstage
	3:    "chozame",    // Sturgeon Shipyard
	4:    "ama",        // Inkblot Art Academy
	5:    "kombu",      // Humpback Pump Track
	6:    "manta",      // Manta Maria
	7:    "hokke",      // Port Mackerel
	8:    "tachiuo",    // Moray Towers
	9:    "engawa",     // Snapper Canal
	10:   "mozuku",     // Kelp Dome
	11:   "bbass",      // Blackbelly Skatepark
	12:   "devon",      // Shellendorf Institute
	13:   "zatou",      // MakoMart
	14:   "hakofugu",   // Walleye Warehouse
	15:   "arowana",    // Arowana Mall
	16:   "mongara",    // Camp Triggerfish
	17:   "shottsuru",  // Piranha Pit
	18:   "ajifry",     // Goby Arena
	19:   "otoro",      // New Albacore Hotel
	20:   "sumeshi",    // Wahoo World
	21:   "anchovy",    // Ancho-V Games
	22:   "mutsugoro",  // Skipper Pavilion
	100:  "mystery_04", // Shifty: Windmill House on the Pearlie
	101:  "mystery_01", // Shifty: Wayslide Cool
	102:  "mystery_02", // Shifty: The Secret of S.P.L.A.T.
	103:  "mystery_03", // Shifty: Goosponge
	105:  "mystery_07", // Shifty: Cannon Fire Pearl
	106:  "mystery_06", // Shifty: Zone of Glass
	107:  "mystery_05", // Shifty: Fancy Spew
	108:  "mystery_09", // Shifty: Grapplink Girl
	109:  "mystery_10", // Shifty: Zappy Longshocking
	110:  "mystery_08", // Shifty: The Bunker Games
	111:  "mystery_11", // Shifty: A Swiftly Tilting Balance
	112:  "mystery_13", // Shifty: The Switches
	113:  "mystery_12", // Shifty: Sweet Valley Tentacles
	114:  "mystery_14", // Shifty: The Bouncey Twins
	115:  "mystery_15", // Shifty: Railway Chillin"
	116:  "mystery_16", // Shifty: Gusher Towns
	117:  "mystery_17", // Shifty: The Maze Dasher
	118:  "mystery_18", // Shifty: Flooders in the Attic
	119:  "mystery_19", // Shifty: The Splat in Our Zones
	120:  "mystery_20", // Shifty: The Ink is Spreading
	121:  "mystery_21", // Shifty: Bridge to Tentaswitchia
	122:  "mystery_22", // Shifty: The Chronicles of Rolonium
	123:  "mystery_23", // Shifty: Furler in the Ashes
	124:  "mystery_24", // Shifty: MC.Princess Diaries
	9999: "mystery",    // Shifty Station
}

// Ability database
//	https://github.com/fetus-hina/stat.ink/blob/master/doc/api-2/post-battle.md#gear-ability
var abilitiesEnum = map[int]string{
	0:   "ink_saver_main",
	1:   "ink_saver_sub",
	2:   "ink_recovery_up",
	3:   "run_speed_up",
	4:   "swim_speed_up",
	5:   "special_charge_up",
	6:   "special_saver",
	7:   "special_power_up",
	8:   "quick_respawn",
	9:   "quick_super_jump",
	10:  "sub_power_up",
	11:  "ink_resistance_up",
	12:  "bomb_defense_up",
	13:  "cold_blooded",
	100: "opening_gambit",
	101: "last_ditch_effort",
	102: "tenacity",
	103: "comeback",
	104: "ninja_squid",
	105: "haunt",
	106: "thermal_ink",
	107: "respawn_punisher",
	108: "ability_doubler",
	109: "stealth_jump",
	110: "object_shredder",
	111: "drop_roller",
	200: "bomb_defense_up_dx",
	201: "main_power_up",
}
