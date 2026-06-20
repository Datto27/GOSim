package adapters

import "fmt"

func init() {
	Register(&MusicAdapter{})
}

// MusicAdapter implements Adapter for the "music" content domain.
type MusicAdapter struct{}

// Type returns "music".
func (a *MusicAdapter) Type() string { return "music" }

// BuildText assembles a descriptive string from a track/album's title,
// artist, genre, mood, and description.
func (a *MusicAdapter) BuildText(fields map[string]any) string {
	title := stringField(fields["title"])
	artist := stringField(fields["artist"])
	genres := stringSlice(fields["genre"])
	moods := stringSlice(fields["mood"])
	description := stringField(fields["description"])

	return fmt.Sprintf(
		"%s by %s. Genre: %s. Mood: %s. Description: %s",
		title, artist, joinOr(genres, "Unknown"), joinOr(moods, "Unknown"), description,
	)
}

// Seeds returns the 25 hardcoded music seed items.
func (a *MusicAdapter) Seeds() []SeedItem {
	return []SeedItem{
		{
			ID:    "music:time-hans-zimmer",
			Label: "Time",
			Fields: map[string]any{
				"title":       "Time",
				"artist":      "Hans Zimmer",
				"genre":       []string{"Soundtrack", "Ambient"},
				"mood":        []string{"Contemplative", "Epic", "Melancholic"},
				"description": "A sweeping orchestral piece from the Inception score that builds slowly into an emotional, time-bending climax, evoking memory, dreams, and the passage of time.",
			},
			Tags: []string{"soundtrack", "ambient", "mind-bending", "memory", "epic"},
		},
		{
			ID:    "music:ok-computer-radiohead",
			Label: "OK Computer",
			Fields: map[string]any{
				"title":       "OK Computer",
				"artist":      "Radiohead",
				"genre":       []string{"Alternative Rock", "Art Rock"},
				"mood":        []string{"Dystopian", "Melancholic", "Anxious"},
				"description": "A landmark album exploring alienation, technology, and consumer culture, with icy soundscapes that paint a portrait of a cold, dehumanized near-future.",
			},
			Tags: []string{"alternative-rock", "dystopian", "melancholic", "technology"},
		},
		{
			ID:    "music:dark-side-of-the-moon",
			Label: "The Dark Side of the Moon",
			Fields: map[string]any{
				"title":       "The Dark Side of the Moon",
				"artist":      "Pink Floyd",
				"genre":       []string{"Progressive Rock"},
				"mood":        []string{"Contemplative", "Existential", "Epic"},
				"description": "A concept album meditating on time, mortality, money, and madness, woven together into a continuous, philosophical sonic journey.",
			},
			Tags: []string{"progressive-rock", "existential", "time", "epic"},
		},
		{
			ID:    "music:discovery-daft-punk",
			Label: "Discovery",
			Fields: map[string]any{
				"title":       "Discovery",
				"artist":      "Daft Punk",
				"genre":       []string{"Electronic", "Dance"},
				"mood":        []string{"Futuristic", "Euphoric"},
				"description": "A genre-defining electronic album blending robotic vocals and funk influences into a glittering vision of the future and the search for identity beneath the machine.",
			},
			Tags: []string{"electronic", "futuristic", "identity"},
		},
		{
			ID:    "music:random-access-memories",
			Label: "Random Access Memories",
			Fields: map[string]any{
				"title":       "Random Access Memories",
				"artist":      "Daft Punk",
				"genre":       []string{"Electronic", "Disco"},
				"mood":        []string{"Nostalgic", "Warm"},
				"description": "A lush homage to late-1970s and '80s music production, blending live instrumentation with electronic textures into a nostalgic celebration of memory and craft.",
			},
			Tags: []string{"electronic", "nostalgic", "memory"},
		},
		{
			ID:    "music:interstellar-hans-zimmer",
			Label: "Interstellar (Soundtrack)",
			Fields: map[string]any{
				"title":       "Interstellar (Soundtrack)",
				"artist":      "Hans Zimmer",
				"genre":       []string{"Soundtrack", "Orchestral"},
				"mood":        []string{"Epic", "Awe-Inspiring", "Emotional"},
				"description": "A towering organ-and-strings score that captures the vastness of space, the bending of time, and the unbreakable pull of family across galaxies.",
			},
			Tags: []string{"soundtrack", "epic", "space", "time"},
		},
		{
			ID:    "music:lateralus-tool",
			Label: "Lateralus",
			Fields: map[string]any{
				"title":       "Lateralus",
				"artist":      "Tool",
				"genre":       []string{"Progressive Metal"},
				"mood":        []string{"Introspective", "Mind-Bending", "Intense"},
				"description": "A dense, polyrhythmic album exploring consciousness, spirals of growth, and the boundaries of perception through layered, hypnotic compositions.",
			},
			Tags: []string{"progressive-metal", "mind-bending", "introspective"},
		},
		{
			ID:    "music:in-rainbows-radiohead",
			Label: "In Rainbows",
			Fields: map[string]any{
				"title":       "In Rainbows",
				"artist":      "Radiohead",
				"genre":       []string{"Alternative Rock"},
				"mood":        []string{"Melancholic", "Intimate", "Introspective"},
				"description": "A warm yet melancholic album of intimate songwriting, layered textures, and emotional vulnerability that feels like a quiet conversation with yourself.",
			},
			Tags: []string{"alternative-rock", "melancholic", "introspective"},
		},
		{
			ID:    "music:fellowship-of-the-ring-howard-shore",
			Label: "The Lord of the Rings: The Fellowship of the Ring (Soundtrack)",
			Fields: map[string]any{
				"title":       "The Lord of the Rings: The Fellowship of the Ring (Soundtrack)",
				"artist":      "Howard Shore",
				"genre":       []string{"Soundtrack", "Orchestral"},
				"mood":        []string{"Epic", "Adventurous", "Heroic"},
				"description": "A sweeping orchestral score built around recurring themes for distant lands and unlikely heroes, capturing the scale of an epic journey into the unknown.",
			},
			Tags: []string{"soundtrack", "epic", "adventure", "fantasy"},
		},
		{
			ID:    "music:dune-hans-zimmer",
			Label: "Dune (Soundtrack)",
			Fields: map[string]any{
				"title":       "Dune (Soundtrack)",
				"artist":      "Hans Zimmer",
				"genre":       []string{"Soundtrack", "Ambient"},
				"mood":        []string{"Epic", "Otherworldly", "Tense"},
				"description": "A vast, percussive, and vocal-driven score that conjures the heat and danger of an alien desert world and the weight of a hero's destiny.",
			},
			Tags: []string{"soundtrack", "epic", "desert", "destiny"},
		},
		{
			ID:    "music:blade-runner-2049-soundtrack",
			Label: "Blade Runner 2049 (Soundtrack)",
			Fields: map[string]any{
				"title":       "Blade Runner 2049 (Soundtrack)",
				"artist":      "Hans Zimmer and Benjamin Wallfisch",
				"genre":       []string{"Soundtrack", "Ambient"},
				"mood":        []string{"Dystopian", "Brooding", "Atmospheric"},
				"description": "A dense wall of synthesizers and droning bass that evokes a rain-soaked dystopian future and the loneliness of searching for identity within it.",
			},
			Tags: []string{"soundtrack", "dystopian", "ambient", "identity"},
		},
		{
			ID:    "music:for-emma-forever-ago",
			Label: "For Emma, Forever Ago",
			Fields: map[string]any{
				"title":       "For Emma, Forever Ago",
				"artist":      "Bon Iver",
				"genre":       []string{"Indie Folk"},
				"mood":        []string{"Melancholic", "Isolated", "Intimate"},
				"description": "A hushed, falsetto-led folk album written in isolation during a harsh winter, brimming with heartbreak, solitude, and quiet beauty.",
			},
			Tags: []string{"indie-folk", "melancholic", "isolation"},
		},
		{
			ID:    "music:currents-tame-impala",
			Label: "Currents",
			Fields: map[string]any{
				"title":       "Currents",
				"artist":      "Tame Impala",
				"genre":       []string{"Psychedelic Rock", "Electronic"},
				"mood":        []string{"Introspective", "Dreamy", "Transformative"},
				"description": "A psychedelic, synth-drenched album about personal transformation, letting go of old identities, and drifting through emotional change.",
			},
			Tags: []string{"psychedelic", "introspective", "identity", "change"},
		},
		{
			ID:    "music:whiplash-soundtrack",
			Label: "Whiplash (Soundtrack)",
			Fields: map[string]any{
				"title":       "Whiplash (Soundtrack)",
				"artist":      "Justin Hurwitz",
				"genre":       []string{"Soundtrack", "Jazz"},
				"mood":        []string{"Intense", "Driving", "Obsessive"},
				"description": "A propulsive jazz score built around relentless drum solos, mirroring the obsessive drive and punishing ambition of a young musician chasing greatness.",
			},
			Tags: []string{"soundtrack", "jazz", "ambition", "obsession"},
		},
		{
			ID:    "music:la-la-land-soundtrack",
			Label: "La La Land (Soundtrack)",
			Fields: map[string]any{
				"title":       "La La Land (Soundtrack)",
				"artist":      "Justin Hurwitz",
				"genre":       []string{"Soundtrack", "Musical"},
				"mood":        []string{"Romantic", "Melancholic", "Dreamy"},
				"description": "A jazzy, bittersweet musical score about chasing dreams in Los Angeles, full of romantic longing and the ache of paths not taken.",
			},
			Tags: []string{"soundtrack", "musical", "romance", "melancholic", "dreams"},
		},
		{
			ID:    "music:kid-a-radiohead",
			Label: "Kid A",
			Fields: map[string]any{
				"title":       "Kid A",
				"artist":      "Radiohead",
				"genre":       []string{"Electronic", "Art Rock"},
				"mood":        []string{"Dystopian", "Futuristic", "Alienated"},
				"description": "An icy, electronic reinvention that abandons traditional rock structure for glitching textures and disembodied vocals, evoking a cold, automated future.",
			},
			Tags: []string{"electronic", "dystopian", "futuristic", "alienation"},
		},
		{
			ID:    "music:plastic-beach-gorillaz",
			Label: "Plastic Beach",
			Fields: map[string]any{
				"title":       "Plastic Beach",
				"artist":      "Gorillaz",
				"genre":       []string{"Alternative", "Hip Hop"},
				"mood":        []string{"Futuristic", "Melancholic", "Satirical"},
				"description": "A concept album set on a synthetic island built from trash, blending genres into a colorful but melancholy portrait of environmental and cultural collapse.",
			},
			Tags: []string{"alternative", "futuristic", "dystopian"},
		},
		{
			ID:    "music:coco-soundtrack",
			Label: "Coco (Soundtrack)",
			Fields: map[string]any{
				"title":       "Coco (Soundtrack)",
				"artist":      "Michael Giacchino and various artists",
				"genre":       []string{"Soundtrack", "Latin"},
				"mood":        []string{"Heartfelt", "Joyful", "Nostalgic"},
				"description": "A vibrant, guitar-driven score and song collection celebrating family, memory, and the way music keeps loved ones alive across generations.",
			},
			Tags: []string{"soundtrack", "family", "memory", "heartfelt"},
		},
		{
			ID:    "music:to-pimp-a-butterfly",
			Label: "To Pimp a Butterfly",
			Fields: map[string]any{
				"title":       "To Pimp a Butterfly",
				"artist":      "Kendrick Lamar",
				"genre":       []string{"Hip Hop", "Jazz Rap"},
				"mood":        []string{"Introspective", "Political", "Intense"},
				"description": "A dense, jazz-infused hip hop album wrestling with identity, fame, and social injustice through sprawling, genre-bending compositions.",
			},
			Tags: []string{"hip-hop", "identity", "introspective"},
		},
		{
			ID:    "music:origin-of-symmetry-muse",
			Label: "Origin of Symmetry",
			Fields: map[string]any{
				"title":       "Origin of Symmetry",
				"artist":      "Muse",
				"genre":       []string{"Alternative Rock", "Progressive Rock"},
				"mood":        []string{"Epic", "Existential", "Dramatic"},
				"description": "A bombastic rock album with operatic vocals and grand orchestration, grappling with cosmic scale, mortality, and existential dread.",
			},
			Tags: []string{"alternative-rock", "epic", "existential", "space"},
		},
		{
			ID:    "music:ghost-in-the-shell-soundtrack",
			Label: "Ghost in the Shell (Soundtrack)",
			Fields: map[string]any{
				"title":       "Ghost in the Shell (Soundtrack)",
				"artist":      "Kenji Kawai",
				"genre":       []string{"Soundtrack", "Ambient"},
				"mood":        []string{"Atmospheric", "Philosophical", "Haunting"},
				"description": "A haunting score blending traditional vocals and electronic textures, underscoring questions of consciousness, identity, and what separates human from machine.",
			},
			Tags: []string{"soundtrack", "identity", "technology", "ambient"},
		},
		{
			ID:    "music:the-suburbs-arcade-fire",
			Label: "The Suburbs",
			Fields: map[string]any{
				"title":       "The Suburbs",
				"artist":      "Arcade Fire",
				"genre":       []string{"Indie Rock"},
				"mood":        []string{"Nostalgic", "Wistful", "Reflective"},
				"description": "A sprawling indie rock album reflecting on childhood, suburban sprawl, and the bittersweet passage from youth into adulthood.",
			},
			Tags: []string{"indie-rock", "nostalgic", "coming-of-age"},
		},
		{
			ID:    "music:stand-by-me-ben-e-king",
			Label: "Stand By Me",
			Fields: map[string]any{
				"title":       "Stand By Me",
				"artist":      "Ben E. King",
				"genre":       []string{"Soul", "Pop"},
				"mood":        []string{"Warm", "Nostalgic", "Reassuring"},
				"description": "A timeless soul ballad about steadfast friendship and devotion, offering quiet comfort in the face of an uncertain world.",
			},
			Tags: []string{"soul", "friendship", "nostalgia", "coming-of-age"},
		},
		{
			ID:    "music:a-moon-shaped-pool",
			Label: "A Moon Shaped Pool",
			Fields: map[string]any{
				"title":       "A Moon Shaped Pool",
				"artist":      "Radiohead",
				"genre":       []string{"Alternative Rock"},
				"mood":        []string{"Melancholic", "Mournful", "Introspective"},
				"description": "A string-laden, somber album about loss and separation, pairing orchestral arrangements with quiet emotional devastation.",
			},
			Tags: []string{"alternative-rock", "melancholic", "loss"},
		},
		{
			ID:    "music:mad-max-fury-road-soundtrack",
			Label: "Mad Max: Fury Road (Soundtrack)",
			Fields: map[string]any{
				"title":       "Mad Max: Fury Road (Soundtrack)",
				"artist":      "Junkie XL",
				"genre":       []string{"Soundtrack", "Electronic"},
				"mood":        []string{"Intense", "Driving", "Epic"},
				"description": "A relentless, percussion-heavy score that fuels a high-speed chase across a scorched desert wasteland, blending orchestra and electronics into pure adrenaline.",
			},
			Tags: []string{"soundtrack", "dystopian", "desert", "epic", "action"},
		},
	}
}
