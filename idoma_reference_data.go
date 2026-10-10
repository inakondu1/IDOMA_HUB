package main

type IdomaHomograph struct {
	Word    string
	Details string
}

type IdomaDialect struct {
	Category string
	Otukpo   string
	Edumoga  string
	Oglewu   string
	Agatu    string
	Ado      string
	Okpoga   string
	Otukpa   string
	Orokam   string
	Owukpa   string
}

type IdomaSpeechSentence struct {
	Number  int
	Idoma   string
	English string
}

var idomaHomographs = []IdomaHomograph{
	{"Akpo", "Ákpó — Packet; Àkpɔ̀ — Silver; Àkpɔ̄ — Rheum/Sleep crust."},
	{"Ebe", "Ɛ̀bɛ́ — Meat; Ɛ̄bɛ̄ — Slap; Ɛ̀bɛ̀ — Look/See/Swep."},
	{"Eche", "Ɛ̀cɛ̀ — World/Outside; Ɛ́cɛ̀ — Bead."},
	{"Eje", "Èjé — Beer; Èjè — Beans; Éjè — Given (name); Ɛ̄jɛ̄ — Tiredness; Ɛ̀jɛ̀ — Cheetah."},
	{"Ele", "Ɛ́lɛ́ — Madness; Ɛ̄lɛ̄ — Piece; Ɛ̀lɛ̄ — Illness; Ɛ̀lɛ̀ — A kind of sacrifice usually performed when a man sleeps with another man's wife."},
	{"Ene", "Ɛ̀nɛ̀ — Four; Ɛ́nɛ́ — Mother; Ɛ̀nɛ̄ — A species of poisonous plant; Ɛ́nɛ̄ — Yesterday."},
	{"Eka", "Ɛ̀kà — Oath/Car; Ɛ̀ká — Ricket; Ékà — Hear say; Ɛ̀kā — Monkey."},
	{"Enya", "Ɛ̀nyá — Disgrace; Ɛ̀ɛ̄nyāà — This one."},
	{"Emu", "Ɛ̀mù — Fire stoke; Ɛ̄mū — Surplus."},
	{"Epa", "Ɛ̀pá — Shortcut; Ɛ̀pà — Two; Ɛ̀pā — Bush cat; Èépá — Scooper."},
	{"Epu", "Ēpū — Leaves; Èpù — Vulture."},
	{"Ekpa", "Ɛ̀kpà — Bag/#200; Ɛ́kpà — Rashes."},
	{"Ebu", "Ébú — Dew; Èbú — Farm hut."},
	{"Ekwu", "Èkwū — Masquerade; Ɛ̀kwū — Fist; Ɛ̀kwú — A species of snake."},
	{"Ewa", "Ɛ̀wá — Bush fowl; Ɛ̀wà — Crowd; Ɛ̄wā — Knife."},
	{"Iche", "Ìcɛ̀ — Game; Ícɛ̀ — Today."},
	{"Ilu", "Ílù — Ant; Ìlū — A species of rat; Ìlù — Groan."},
	{"Ikpo", "Ìkpó — Leg; Īkpō — Seed/Grain."},
	{"Ikwu", "Íkwù — Crocodile; Ìkwū — Cry/Death."},
	{"Igwu", "Ìgwú — Meaning not supplied in the source; Ìgwù — Guinea corn; Ìgwū — Hunchback; Ìgwù — Group of persons."},
	{"Inyi", "Īnyī — Dirt; Ìnyì — Loss of weight."},
	{"Ili", "Īlī — Clothes/Cloth; Ílì — Vein."},
	{"Ipi", "Ìpí — Wound; Ípī — Odour/Smell."},
	{"Ije", "Ìjé — Guinea fowl; Ìjè — Money; Ìjē — Fats/Beating/Song/Dance."},
	{"Ochu", "Òcù — Aroma/Scent; Ɔ̀̀cú — Name (feminine)."},
	{"Odo", "Òdò — Heart/Yellow; Òdó — Chain for binding humans and animals; Ɔ̀̀dɔ̀ — Wall; Ódò — Name (masculine); Ódó — Wild palm."},
	{"Ofu", "Ófù — Stroll; Òfù — Twenty; Òfū — Cold; Ɔ̀̀fú — Strength/Power; Ɔ̀́fú — Fever."},
	{"Olo", "Òló — Periwinkle; Ɔ̀̀lɔ̀ — Gills; Ɔ̀́lɔ́ — Deformity."},
	{"Oko", "Ɔ̀̀kɔ̄ — Cough; Ɔ̀̀kɔ̀ — Neck/Prayer; Ɔ̀́kɔ̀ — Canoe; Òkō — Name (masculine); Ókò — Parrot."},
	{"Oha", "Ɔ̀̀há — Next; Ɔ̀̀hà — Garmalin 20; Ɔ̀̄hā — Blessing; Òōhà — Tease."},
	{"Okpa", "Ɔ̀̀kpá — Book/Skin/River/Oath; Ɔ̀̀kpà — Bambara nut cakes; Ɔ̀̄kpā — Friend; Ɔ̀́kpà — Spear."},
	{"Oba", "Ɔ̀̀bá — Husband; Ɔ̀́bā — Rain shadow; Ɔ̀̀bà — Drum (Edumoga dialect)."},
	{"Obu", "Ɔ̀̀bū — Front/Ahead/Forward; Òbū — Darkness."},
	{"Ocha", "Ɔ̀̀cá — Squirrel; Ɔ̀̀cà — Zebra (Otina is only a description)."},
	{"Ogba", "Ɔ̀̀gbà — Line/Queue; Ɔ̀́gbā — Rainbow."},
	{"Ola", "Ɔ̀̀lá — Fire/Light; Ɔ̀́lá — Sales; Ɔ̀̀là — A swelling usually around the thigh gap."},
	{"Okwu", "Òkwū — Corpse; Ɔ̀́kwú — Pelvic girdle; Ɔ̀̀kwù — Buttocks; Ɔ̀̄kwū — Weight gain."},
	{"Otu", "Òtú — Night; Òtù — Response; Ɔ̀tū — Chest/Along."},
	{"Owa", "Ɔ̀̀wà — A species of snake; Ɔ̀́wá — Peer group/Light complexion/Smith."},
	{"Owe", "Ōwē — Suffering/Hardship; Ɔ̀̀wɛ̀ — Door/Way/Road."},
	{"Owo", "Ɔ̀̀wɔ̀ — Rain/Destiny; Ɔ̀́wɔ́ — Group walk; Ōwō — Gunpowder."},
	{"Owu", "Òwú — Cotton/Thread; Òwù — Wind/Air/Breath; Ɔ̄̄wū — Cover."},
	{"Ubu", "Ùbù — Tripe (towel meat); Úbú — Depth of a river."},
	{"Udo", "Ùdɔ̀ — Gold; Ùdɔ́ — Bat; Údɔ̀ — Slack."},
	{"Le", "Lé — Eat; Lɛ̀ — Have/Marry."},
}

var idomaDialects = []IdomaDialect{
	{
		Category: "Food",
		Otukpo:   "Òdlé", Edumoga: "Òdlé", Oglewu: "Òdlé", Agatu: "Òdlé",
		Ado: "Òjīré", Okpoga: "Òjīré", Otukpa: "Òglé", Orokam: "Òjīré", Owukpa: "Òjīré",
	},
	{
		Category: "Okra",
		Otukpo:   "Īkp'ɔ̄hɔ̄", Edumoga: "Īkp'ɔ̄lɔ̄", Oglewu: "Īkp'ɔ̄hɔ̄", Agatu: "Ɔ̀hɔ̄ɔ̀dù",
		Ado: "Ìdù", Okpoga: "Ìgbíìdù", Otukpa: "Ìgbíìdù", Orokam: "Ìgbíìdù", Owukpa: "Ìgbíìdù",
	},
	{
		Category: "Clothes",
		Otukpo:   "Īlī", Edumoga: "Īlī", Oglewu: "Īlī", Agatu: "Īlī",
		Ado: "Īrī", Okpoga: "Īrī", Otukpa: "Īlī", Orokam: "Īrī", Owukpa: "Īrī",
	},
	{
		Category: "Play",
		Otukpo:   "Ìjā", Edumoga: "Ìlɛ̄", Oglewu: "Ìjā", Agatu: "Ìjā",
		Ado: "Ìrɛ̄", Okpoga: "Ìrɛ̄", Otukpa: "Ìrɛ̄", Orokam: "Ìrɛ̄", Owukpa: "Ìrɛ̄",
	},
	{
		Category: "Cutlass",
		Otukpo:   "Úkpákwū", Edumoga: "Ìgbáàndlɛ̀", Oglewu: "Úkpákwū / Ɔ̀gbáàndlɛ̀ / Ɔ̀pyà", Agatu: "Ɔ̀mpyà",
		Ado: "Not supplied", Okpoga: "Ìgbáàndrɛ̀", Otukpa: "Ɔ̀gbáànglɛ̀", Orokam: "Ùgbáàndrɛ̀ / Ɔ̀pyà", Owukpa: "Ìgbáàndrɛ̀",
	},
	{
		Category: "Salt",
		Otukpo:   "Ɔ̀mā", Edumoga: "Ɔ̀mā", Oglewu: "Ɔ̀mā", Agatu: "Ɔ̀mā",
		Ado: "Not supplied", Okpoga: "Ɔ̀mā", Otukpa: "Ɔ̀mā", Orokam: "Íkérìké", Owukpa: "Ɔ̀mā",
	},
	{
		Category: "Rat",
		Otukpo:   "Ìfú", Edumoga: "Ìfú", Oglewu: "Ìfú", Agatu: "Ìyú",
		Ado: "Not supplied", Okpoga: "Ìfú", Otukpa: "Ìfú", Orokam: "Íkérékwū", Owukpa: "Ìfú",
	},
}

var idomaSpeechWork = []IdomaSpeechSentence{
	{Number: 1, Idoma: `Àgbó yɔ̄ gw'ābɔ̄ l'ùwá`, English: "Agbo is clapping for them."},
	{Number: 2, Idoma: `Òkō yɔ̄ gw'ɛ̀b'īnù`, English: "Okoh is sweeping the room."},
	{Number: 3, Idoma: `Ɛ́nɛ́ yɔ̄ y'ódlé`, English: "Ene is cooking."},
	{Number: 4, Idoma: `Àbá yɔ̄ n'ùnù túwā`, English: "Aba is fighting them."},
	{Number: 5, Idoma: `Àgbénú yɔ̄ t'ɔ́kpá ɛ̀cɛ̀`, English: "Aggenu is writing outside."},
	{Number: 6, Idoma: `Ìdùh yɔ̄ j'ɔ́kpá ín'ógwùtá`, English: "Iduh is reading a book in the bedroom."},
	{Number: 7, Idoma: `Ádāàm yɔ̄ c'ɔ̄nū d'àlɔ̀`, English: "My father is angry with us."},
	{Number: 8, Idoma: `Ɛ́nɛ́ɛ̀m yɔ̄ gw'íjē ìgbīhī ɔ̄lɛ́`, English: "My mother is singing behind the house."},
	{Number: 9, Idoma: `Ùwá báā hījémā k'ódéè n'ùwá yá āà`, English: "They are regretting what they did."},
}
