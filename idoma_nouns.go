package main

func idomaNounWords() []IdomaWord {
	const source = "Assumpta Ethel Enuwa Umele"
	const dialect = "Not specified in source"

	noun := func(word, meaning, category string) IdomaWord {
		return IdomaWord{
			Word:     word,
			Meaning:  meaning,
			Category: "Noun · " + category,
			Dialect:  dialect,
			Source:   source,
		}
	}

	return []IdomaWord{
		noun("Ɔ̀̀lɛ́", "House/Home", "Home"),
		noun("Ín'ōgwùtá", "Bedroom", "Rooms"),
		noun("Ín'ōy'ódlé/Ɛ̀g'ògwɔ́lá", "Kitchen", "Rooms"),
		noun("Ìn'òyājɛ̀", "Sittingroom", "Rooms"),
		noun("Ìn'ògw'īyē", "Bathroom", "Rooms"),
		noun("Ùkpɔ́", "Toilet", "Rooms"),
		noun("Àgōdō", "Bed", "Furniture and bedding"),
		noun("Ɔ̀̀d'ōlɛyícɔ̀ɔ́", "Pillow", "Furniture and bedding"),
		noun("Ōkwùntú", "Blanket", "Furniture and bedding"),
		noun("Ēkōb'ɔ́ɔ̀dɔ̀", "Wall clock", "Household items"),
		noun("Ɔ̀c'ōyājɛ̀", "Chair", "Furniture"),
		noun("Ìgbé/ Ìkpàtì", "Box", "Household items"),
		noun("Èkòb'ábɔ̄", "Wrist watch", "Accessories"),
		noun("Àdàbā/Áblákpá", "Foot wear", "Clothing and footwear"),
		noun("Ōhílī", "Wrapper", "Clothing and footwear"),
		noun("Úwé", "Blouse/Shirt", "Clothing and footwear"),
		noun("Ɔ̀mù", "Socks", "Clothing and footwear"),
		noun("Ɔ̀̀m'ābɔ̄", "Hand gloves", "Clothing and footwear"),
		noun("Ògìgò", "Mirror /Glass", "Household items"),
		noun("Àdíkò", "Scarf/Head tie", "Clothing and accessories"),
		noun("Ùw'ámɛ̄", "Brazier", "Household items"),
		noun("Ìkápà", "Pant", "Clothing and footwear"),
		noun("Ìkāké", "Trousers/Knicker", "Clothing and footwear"),
		noun("Ínú", "Room", "Rooms"),
		noun("Ùlè", "Window", "Household items"),
	}
}
