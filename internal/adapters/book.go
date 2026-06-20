package adapters

import "fmt"

func init() {
	Register(&BookAdapter{})
}

// BookAdapter implements Adapter for the "book" content domain.
type BookAdapter struct{}

// Type returns "book".
func (a *BookAdapter) Type() string { return "book" }

// BuildText assembles a descriptive string from a book's title, author,
// genre, and synopsis.
func (a *BookAdapter) BuildText(fields map[string]any) string {
	title := stringField(fields["title"])
	author := stringField(fields["author"])
	genres := stringSlice(fields["genre"])
	synopsis := stringField(fields["synopsis"])

	return fmt.Sprintf(
		"%s by %s. Genre: %s. Synopsis: %s",
		title, author, joinOr(genres, "Unknown"), synopsis,
	)
}

// Seeds returns the 25 hardcoded book seed items.
func (a *BookAdapter) Seeds() []SeedItem {
	return []SeedItem{
		{
			ID:    "book:recursion",
			Label: "Recursion",
			Fields: map[string]any{
				"title":    "Recursion",
				"author":   "Blake Crouch",
				"genre":    []string{"Sci-Fi", "Thriller"},
				"synopsis": "A detective investigating a rash of false-memory cases and a neuroscientist who built a machine to record memories discover a technology that lets people relive and overwrite their pasts, with reality-shattering consequences.",
			},
			Tags: []string{"sci-fi", "thriller", "memory", "identity", "mind-bending"},
		},
		{
			ID:    "book:dark-matter",
			Label: "Dark Matter",
			Fields: map[string]any{
				"title":    "Dark Matter",
				"author":   "Blake Crouch",
				"genre":    []string{"Sci-Fi", "Thriller"},
				"synopsis": "A physicist is abducted into a parallel version of his life and must navigate an infinite landscape of alternate realities to find his way back to his true family.",
			},
			Tags: []string{"sci-fi", "thriller", "identity", "mind-bending"},
		},
		{
			ID:    "book:nineteen-eighty-four",
			Label: "1984",
			Fields: map[string]any{
				"title":    "1984",
				"author":   "George Orwell",
				"genre":    []string{"Dystopian", "Political Fiction"},
				"synopsis": "In a totalitarian state under constant surveillance, a low-ranking bureaucrat begins to question the regime's control over truth, memory, and thought itself.",
			},
			Tags: []string{"dystopian", "surveillance", "totalitarian"},
		},
		{
			ID:    "book:brave-new-world",
			Label: "Brave New World",
			Fields: map[string]any{
				"title":    "Brave New World",
				"author":   "Aldous Huxley",
				"genre":    []string{"Dystopian", "Science Fiction"},
				"synopsis": "In a engineered future society built on pleasure and conformity, a few individuals begin to question whether comfort and control are worth the cost of freedom and identity.",
			},
			Tags: []string{"dystopian", "society", "identity"},
		},
		{
			ID:    "book:dune-novel",
			Label: "Dune",
			Fields: map[string]any{
				"title":    "Dune",
				"author":   "Frank Herbert",
				"genre":    []string{"Sci-Fi", "Epic"},
				"synopsis": "The son of a noble family is thrust into a war for control of a harsh desert planet that produces the universe's most valuable substance, as he discovers his own prophesied destiny.",
			},
			Tags: []string{"sci-fi", "epic", "desert", "destiny", "adventure"},
		},
		{
			ID:    "book:fellowship-of-the-ring-novel",
			Label: "The Lord of the Rings: The Fellowship of the Ring",
			Fields: map[string]any{
				"title":    "The Lord of the Rings: The Fellowship of the Ring",
				"author":   "J.R.R. Tolkien",
				"genre":    []string{"Fantasy", "Epic"},
				"synopsis": "A young hobbit inherits a ring of immense power and sets out with a fellowship of companions on a perilous quest across Middle-earth to destroy it before a dark lord can reclaim it.",
			},
			Tags: []string{"fantasy", "epic", "adventure", "journey"},
		},
		{
			ID:    "book:the-hobbit",
			Label: "The Hobbit",
			Fields: map[string]any{
				"title":    "The Hobbit",
				"author":   "J.R.R. Tolkien",
				"genre":    []string{"Fantasy", "Adventure"},
				"synopsis": "A reluctant hobbit is swept into an unexpected journey with a band of dwarves to reclaim a stolen treasure guarded by a dragon, discovering courage he never knew he had.",
			},
			Tags: []string{"fantasy", "adventure", "coming-of-age", "journey"},
		},
		{
			ID:    "book:project-hail-mary",
			Label: "Project Hail Mary",
			Fields: map[string]any{
				"title":    "Project Hail Mary",
				"author":   "Andy Weir",
				"genre":    []string{"Sci-Fi"},
				"synopsis": "A lone astronaut wakes up with no memory on a desperate solo mission to save humanity from extinction, piecing together the truth through ingenuity, science, and an unlikely friendship.",
			},
			Tags: []string{"sci-fi", "space", "survival", "hope"},
		},
		{
			ID:    "book:the-three-body-problem",
			Label: "The Three-Body Problem",
			Fields: map[string]any{
				"title":    "The Three-Body Problem",
				"author":   "Liu Cixin",
				"genre":    []string{"Sci-Fi", "Epic"},
				"synopsis": "A secret military project sends signals into space in search of extraterrestrial life, setting off a chain of events that leads to first contact with an alien civilization and a threat to humanity's future.",
			},
			Tags: []string{"sci-fi", "epic", "space", "first-contact"},
		},
		{
			ID:    "book:slaughterhouse-five",
			Label: "Slaughterhouse-Five",
			Fields: map[string]any{
				"title":    "Slaughterhouse-Five",
				"author":   "Kurt Vonnegut",
				"genre":    []string{"Sci-Fi", "Satire"},
				"synopsis": "A soldier becomes unstuck in time, experiencing the events of his life and a brutal wartime bombing out of order, as he grapples with fate, free will, and the absurdity of war.",
			},
			Tags: []string{"sci-fi", "time", "war", "existential"},
		},
		{
			ID:    "book:the-time-travelers-wife",
			Label: "The Time Traveler's Wife",
			Fields: map[string]any{
				"title":    "The Time Traveler's Wife",
				"author":   "Audrey Niffenegger",
				"genre":    []string{"Sci-Fi", "Romance"},
				"synopsis": "A man with a genetic disorder that causes him to time-travel uncontrollably falls in love with a woman who must learn to live with a relationship scattered across past, present, and future.",
			},
			Tags: []string{"sci-fi", "romance", "time", "melancholic"},
		},
		{
			ID:    "book:never-let-me-go",
			Label: "Never Let Me Go",
			Fields: map[string]any{
				"title":    "Never Let Me Go",
				"author":   "Kazuo Ishiguro",
				"genre":    []string{"Dystopian", "Literary Fiction"},
				"synopsis": "Three friends grow up in an idyllic boarding school, slowly uncovering the haunting truth about their purpose and the limited future that awaits them.",
			},
			Tags: []string{"dystopian", "melancholic", "identity", "memory"},
		},
		{
			ID:    "book:the-road",
			Label: "The Road",
			Fields: map[string]any{
				"title":    "The Road",
				"author":   "Cormac McCarthy",
				"genre":    []string{"Post-Apocalyptic", "Literary Fiction"},
				"synopsis": "A father and son journey across a devastated, ash-covered landscape, clinging to love and survival in a world stripped of nearly everything else.",
			},
			Tags: []string{"post-apocalyptic", "survival", "dystopian"},
		},
		{
			ID:    "book:fahrenheit-451",
			Label: "Fahrenheit 451",
			Fields: map[string]any{
				"title":    "Fahrenheit 451",
				"author":   "Ray Bradbury",
				"genre":    []string{"Dystopian", "Science Fiction"},
				"synopsis": "In a future where books are outlawed and burned, a fireman who destroys them begins to question a society that has traded thought and memory for distraction and conformity.",
			},
			Tags: []string{"dystopian", "censorship", "society"},
		},
		{
			ID:    "book:the-fault-in-our-stars",
			Label: "The Fault in Our Stars",
			Fields: map[string]any{
				"title":    "The Fault in Our Stars",
				"author":   "John Green",
				"genre":    []string{"Young Adult", "Romance"},
				"synopsis": "Two teenagers with cancer fall in love and confront mortality, friendship, and the search for meaning in the time they have together.",
			},
			Tags: []string{"young-adult", "romance", "melancholic", "coming-of-age"},
		},
		{
			ID:    "book:the-perks-of-being-a-wallflower-novel",
			Label: "The Perks of Being a Wallflower",
			Fields: map[string]any{
				"title":    "The Perks of Being a Wallflower",
				"author":   "Stephen Chbosky",
				"genre":    []string{"Young Adult", "Drama"},
				"synopsis": "A shy, observant teenager writes letters about his freshman year of high school, navigating new friendships, first love, and painful memories he is only beginning to understand.",
			},
			Tags: []string{"young-adult", "coming-of-age", "melancholic", "friendship"},
		},
		{
			ID:    "book:norwegian-wood",
			Label: "Norwegian Wood",
			Fields: map[string]any{
				"title":    "Norwegian Wood",
				"author":   "Haruki Murakami",
				"genre":    []string{"Literary Fiction", "Romance"},
				"synopsis": "A college student in 1960s Tokyo reflects on love, loss, and the fragile mental health of the people closest to him, in a quiet, melancholic meditation on memory and grief.",
			},
			Tags: []string{"literary-fiction", "melancholic", "loss", "coming-of-age"},
		},
		{
			ID:    "book:kafka-on-the-shore",
			Label: "Kafka on the Shore",
			Fields: map[string]any{
				"title":    "Kafka on the Shore",
				"author":   "Haruki Murakami",
				"genre":    []string{"Magical Realism"},
				"synopsis": "A runaway teenager and an elderly man with a mysterious gift follow intertwining, dreamlike paths that blur the line between memory, fate, and the world of the unconscious.",
			},
			Tags: []string{"magical-realism", "mind-bending", "identity", "dreams"},
		},
		{
			ID:    "book:the-shadow-of-the-wind",
			Label: "The Shadow of the Wind",
			Fields: map[string]any{
				"title":    "The Shadow of the Wind",
				"author":   "Carlos Ruiz Zafon",
				"genre":    []string{"Mystery", "Literary Fiction"},
				"synopsis": "A young boy discovers a mysterious novel and sets out to find its author, uncovering a tragic history of love and betrayal hidden within the shadowy streets of postwar Barcelona.",
			},
			Tags: []string{"mystery", "atmospheric", "literary-fiction"},
		},
		{
			ID:    "book:ready-player-one",
			Label: "Ready Player One",
			Fields: map[string]any{
				"title":    "Ready Player One",
				"author":   "Ernest Cline",
				"genre":    []string{"Sci-Fi", "Adventure"},
				"synopsis": "In a bleak future where most people escape into a vast virtual reality, a teenager hunts for a hidden treasure left behind by the simulation's creator, racing through a world built from nostalgic pop culture.",
			},
			Tags: []string{"sci-fi", "virtual-reality", "nostalgia", "adventure"},
		},
		{
			ID:    "book:ghost-in-the-shell-manga",
			Label: "Ghost in the Shell",
			Fields: map[string]any{
				"title":    "Ghost in the Shell",
				"author":   "Masamune Shirow",
				"genre":    []string{"Sci-Fi", "Manga"},
				"synopsis": "A cyborg counter-terrorism agent hunts a mysterious hacker while questioning the nature of consciousness, identity, and the soul in a world where minds can be uploaded and bodies replaced.",
			},
			Tags: []string{"sci-fi", "identity", "technology", "philosophical"},
		},
		{
			ID:    "book:cloud-atlas",
			Label: "Cloud Atlas",
			Fields: map[string]any{
				"title":    "Cloud Atlas",
				"author":   "David Mitchell",
				"genre":    []string{"Sci-Fi", "Epic"},
				"synopsis": "Six interconnected stories spanning centuries and continents trace how the choices of one soul echo across time, weaving a sweeping meditation on fate, identity, and rebirth.",
			},
			Tags: []string{"sci-fi", "epic", "time", "identity"},
		},
		{
			ID:    "book:the-midnight-library",
			Label: "The Midnight Library",
			Fields: map[string]any{
				"title":    "The Midnight Library",
				"author":   "Matt Haig",
				"genre":    []string{"Speculative Fiction"},
				"synopsis": "Between life and death lies a library of infinite books, each one a different version of the life a woman could have lived, as she searches for the one worth living.",
			},
			Tags: []string{"speculative-fiction", "identity", "parallel-lives", "hopeful"},
		},
		{
			ID:    "book:the-body-stephen-king",
			Label: "The Body",
			Fields: map[string]any{
				"title":    "The Body",
				"author":   "Stephen King",
				"genre":    []string{"Novella", "Drama"},
				"synopsis": "Four young friends hike along railroad tracks to find the body of a missing boy, sharing a journey that marks the bittersweet end of childhood.",
			},
			Tags: []string{"drama", "coming-of-age", "friendship", "nostalgia"},
		},
		{
			ID:    "book:world-war-z",
			Label: "World War Z",
			Fields: map[string]any{
				"title":    "World War Z",
				"author":   "Max Brooks",
				"genre":    []string{"Post-Apocalyptic", "Horror"},
				"synopsis": "An oral history of a global zombie pandemic, told through the voices of survivors from every corner of the world, chronicling humanity's collapse and fight for survival.",
			},
			Tags: []string{"post-apocalyptic", "survival", "dystopian", "epic"},
		},
	}
}
