package main

import "strings"

const idomaCultureReferenceNotes = `CURATED IDOMA CULTURAL REFERENCE NOTES

Use these notes carefully. They combine existing IDOMA HUB history content with named external references. Distinguish documented information from oral traditions, community beliefs, and practices that may vary.

1. HISTORY AND ORIGINS
IDOMA HUB's History page explains that Idoma origin traditions are preserved through oral history, family genealogies, and community traditions. Some communities connect their histories with Apa and the wider Kwararafa region. The page also recognises that Idoma identity developed through different groups and migrations and that communities may preserve different accounts. Do not claim that one origin account is universally accepted.

2. ALEKWU AND TRADITIONAL BELIEFS
Research by Guda Frederick Nda and co-authors, published in the Benue Journal of Sociology, examines Alekwu as an ancestral tradition associated in some Idoma communities with conduct and family morality.
Present this as a documented cultural belief or tradition, not as a scientifically established supernatural claim. Do not assume every Idoma person or community follows the same practices.
Source: The Alekwu Tradition and Marital Stability among the Idoma Ethnic Group in Benue State, Central Nigeria.
https://www.bsum.edu.ng/journals/jsoc/vol12n2/article.php?id=150

Research by Emmanuel C. Anizoba and Edache Monday Johnson also discusses changes in Idoma traditional religious and cultural practices, including ancestor veneration, burial rites, widowhood practices, and naming ceremonies. Their study considers the influence of Christianity. Do not present historical practices as universal current practices.
Source: Patterns of Traditional Religious and Cultural Practices of the Idoma People of Nigeria (2021).
https://www.ajol.info/index.php/ft/article/view/216900

3. CLOTHING
Nigeria's Federal Ministry of Information and National Orientation describes red-and-black striped attire as associated with the Idoma people. Do not invent symbolic meanings for the individual colours without a reliable source.
Source: Federal Ministry of Information and National Orientation, Culture.
https://fmino.gov.ng/culture/culture/

4. CONTEMPORARY CULTURAL CELEBRATIONS
Nigeria's National Institute for Cultural Orientation documents the Idoma International Carnival in Otukpo, formerly known as the Agila Social and Economic Carnival. Its description includes cultural parades, masquerade displays, music, and dance.
Source: National Institute for Cultural Orientation, Idoma International Carnival.
https://nico.gov.ng/idoma-international-carnival/

5. DISTINGUISH IDOMA AND IGEDE TRADITIONS
The National Institute for Cultural Orientation identifies Igede Agba as the New Yam Festival of the Igede people of Oju and Obi. Do not describe Igede Agba as an Idoma festival.
Source: National Institute for Cultural Orientation, Igede Agba Festival.
https://nico.gov.ng/igede-agba-festival/

6. IMPORTANT LIMITATIONS
These references do not establish every detail of Idoma traditional leadership, every community's festival calendar, or all local customs. In particular, do not confidently invent details about named institutions, festival dates, rituals, or the meanings of cultural symbols.
If a question concerns a specific community, ask which community when necessary or explain that the available references do not establish the answer.
Do not invent sources or citations. When these notes support an answer, include a short Sources section containing the relevant source titles and URLs above.
`

func idomaCultureReferenceContext(question string) string {
	q := strings.ToLower(strings.TrimSpace(question))
	if q == "" {
		return ""
	}

	keywords := []string{
		"culture", "tradition", "history", "origin", "migration",
		"alekwu", "festival", "attire", "clothing", "masquerade",
		"belief", "ancestor", "spiritual", "kwararafa", "apa",
		"carnival", "ochi'idoma", "traditional ruler", "custom",
		"burial", "widowhood", "naming ceremony", "food", "cuisine",
		"new yam", "eje agba", "eje alekwu", "oje", "community tradition",
	}

	for _, keyword := range keywords {
		if strings.Contains(q, keyword) {
			return idomaCultureReferenceNotes
		}
	}

	return ""
}
