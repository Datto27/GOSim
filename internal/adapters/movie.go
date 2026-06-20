package adapters

import "fmt"

func init() {
	Register(&MovieAdapter{})
}

// MovieAdapter implements Adapter for the "movie" content domain.
type MovieAdapter struct{}

// Type returns "movie".
func (a *MovieAdapter) Type() string { return "movie" }

// BuildText assembles a descriptive string from a movie's title, year,
// genre, cast, and plot.
func (a *MovieAdapter) BuildText(fields map[string]any) string {
	title := stringField(fields["title"])
	year := intField(fields["year"])
	genres := stringSlice(fields["genre"])
	cast := stringSlice(fields["cast"])
	plot := stringField(fields["plot"])

	return fmt.Sprintf(
		"%s (%d). Genre: %s. Starring: %s. Plot: %s",
		title, year, joinOr(genres, "Unknown"), joinOr(cast, "Unknown"), plot,
	)
}

// Seeds returns the 25 hardcoded movie seed items.
func (a *MovieAdapter) Seeds() []SeedItem {
	return []SeedItem{
		{
			ID:    "movie:inception",
			Label: "Inception",
			Fields: map[string]any{
				"title": "Inception",
				"year":  2010,
				"genre": []string{"Sci-Fi", "Thriller"},
				"cast":  []string{"Leonardo DiCaprio", "Joseph Gordon-Levitt", "Elliot Page"},
				"plot":  "A skilled thief who steals secrets from the subconscious during dreams is offered a chance to have his criminal past erased if he can plant an idea in a target's mind instead of stealing one.",
			},
			Tags: []string{"sci-fi", "thriller", "mind-bending", "dreams", "memory", "identity"},
		},
		{
			ID:    "movie:the-matrix",
			Label: "The Matrix",
			Fields: map[string]any{
				"title": "The Matrix",
				"year":  1999,
				"genre": []string{"Sci-Fi", "Action"},
				"cast":  []string{"Keanu Reeves", "Laurence Fishburne", "Carrie-Anne Moss"},
				"plot":  "A computer hacker learns that the world he knows is a simulated reality created by machines, and joins a rebellion to free humanity from it.",
			},
			Tags: []string{"sci-fi", "action", "simulated-reality", "identity", "mind-bending"},
		},
		{
			ID:    "movie:interstellar",
			Label: "Interstellar",
			Fields: map[string]any{
				"title": "Interstellar",
				"year":  2014,
				"genre": []string{"Sci-Fi", "Drama"},
				"cast":  []string{"Matthew McConaughey", "Anne Hathaway", "Jessica Chastain"},
				"plot":  "A team of explorers travels through a wormhole in space in an attempt to ensure humanity's survival, confronting the relativity of time and the bonds of family along the way.",
			},
			Tags: []string{"sci-fi", "drama", "epic", "space", "time"},
		},
		{
			ID:    "movie:blade-runner-2049",
			Label: "Blade Runner 2049",
			Fields: map[string]any{
				"title": "Blade Runner 2049",
				"year":  2017,
				"genre": []string{"Sci-Fi", "Neo-Noir"},
				"cast":  []string{"Ryan Gosling", "Harrison Ford", "Ana de Armas"},
				"plot":  "A young blade runner discovers a secret that could plunge society into chaos, leading him to seek out a former blade runner who has been missing for thirty years, all while questioning what it means to be human.",
			},
			Tags: []string{"sci-fi", "dystopian", "identity", "neo-noir", "ambient"},
		},
		{
			ID:    "movie:eternal-sunshine-of-the-spotless-mind",
			Label: "Eternal Sunshine of the Spotless Mind",
			Fields: map[string]any{
				"title": "Eternal Sunshine of the Spotless Mind",
				"year":  2004,
				"genre": []string{"Romance", "Sci-Fi"},
				"cast":  []string{"Jim Carrey", "Kate Winslet"},
				"plot":  "After a painful breakup, a man undergoes a procedure to erase memories of his ex-girlfriend from his mind, only to discover what he truly cherished as the memories fade.",
			},
			Tags: []string{"romance", "sci-fi", "memory", "melancholic", "identity"},
		},
		{
			ID:    "movie:the-fellowship-of-the-ring",
			Label: "The Lord of the Rings: The Fellowship of the Ring",
			Fields: map[string]any{
				"title": "The Lord of the Rings: The Fellowship of the Ring",
				"year":  2001,
				"genre": []string{"Fantasy", "Adventure"},
				"cast":  []string{"Elijah Wood", "Ian McKellen", "Viggo Mortensen"},
				"plot":  "A young hobbit and a fellowship of companions set out on a perilous journey to destroy a powerful ring before it falls into the hands of a dark lord bent on conquering the world.",
			},
			Tags: []string{"fantasy", "adventure", "epic", "journey"},
		},
		{
			ID:    "movie:dune",
			Label: "Dune",
			Fields: map[string]any{
				"title": "Dune",
				"year":  2021,
				"genre": []string{"Sci-Fi", "Adventure"},
				"cast":  []string{"Timothee Chalamet", "Rebecca Ferguson", "Zendaya"},
				"plot":  "The son of a noble family is thrust into a war for control of a desert planet that holds the key to humanity's most valuable resource, discovering his own destiny along the way.",
			},
			Tags: []string{"sci-fi", "adventure", "epic", "desert", "destiny"},
		},
		{
			ID:    "movie:children-of-men",
			Label: "Children of Men",
			Fields: map[string]any{
				"title": "Children of Men",
				"year":  2006,
				"genre": []string{"Sci-Fi", "Drama"},
				"cast":  []string{"Clive Owen", "Julianne Moore", "Michael Caine"},
				"plot":  "In a near future where humanity has become infertile, a disillusioned bureaucrat is tasked with escorting a miraculously pregnant woman to a sanctuary, becoming humanity's last hope.",
			},
			Tags: []string{"sci-fi", "drama", "dystopian", "survival"},
		},
		{
			ID:    "movie:the-shawshank-redemption",
			Label: "The Shawshank Redemption",
			Fields: map[string]any{
				"title": "The Shawshank Redemption",
				"year":  1994,
				"genre": []string{"Drama"},
				"cast":  []string{"Tim Robbins", "Morgan Freeman"},
				"plot":  "A banker wrongly convicted of murder forms an unlikely friendship with a fellow inmate over decades, finding hope and redemption within the walls of a brutal prison.",
			},
			Tags: []string{"drama", "hope", "redemption", "friendship"},
		},
		{
			ID:    "movie:pans-labyrinth",
			Label: "Pan's Labyrinth",
			Fields: map[string]any{
				"title": "Pan's Labyrinth",
				"year":  2006,
				"genre": []string{"Fantasy", "Drama"},
				"cast":  []string{"Ivana Baquero", "Sergi Lopez", "Doug Jones"},
				"plot":  "A young girl escapes the brutal reality of fascist Spain by retreating into a dark fairy-tale world, where she is given three dangerous tasks to complete by a mysterious faun.",
			},
			Tags: []string{"fantasy", "drama", "dark-fairy-tale", "coming-of-age"},
		},
		{
			ID:    "movie:spirited-away",
			Label: "Spirited Away",
			Fields: map[string]any{
				"title": "Spirited Away",
				"year":  2001,
				"genre": []string{"Animation", "Fantasy"},
				"cast":  []string{"Rumi Hiiragi", "Miyu Irino"},
				"plot":  "A young girl wanders into a magical world ruled by spirits and must work in a bathhouse to free herself and her parents, who have been transformed by a witch.",
			},
			Tags: []string{"animation", "fantasy", "coming-of-age", "adventure"},
		},
		{
			ID:    "movie:the-grand-budapest-hotel",
			Label: "The Grand Budapest Hotel",
			Fields: map[string]any{
				"title": "The Grand Budapest Hotel",
				"year":  2014,
				"genre": []string{"Comedy", "Drama"},
				"cast":  []string{"Ralph Fiennes", "Tony Revolori"},
				"plot":  "The adventures of a legendary concierge at a famous European hotel between the two world wars, and the lobby boy who becomes his trusted protege, told with whimsical, meticulous style.",
			},
			Tags: []string{"comedy", "drama", "whimsical"},
		},
		{
			ID:    "movie:whiplash",
			Label: "Whiplash",
			Fields: map[string]any{
				"title": "Whiplash",
				"year":  2014,
				"genre": []string{"Drama", "Music"},
				"cast":  []string{"Miles Teller", "J.K. Simmons"},
				"plot":  "A young jazz drummer enrolls at a cutthroat music conservatory, pushing himself to the brink of physical and emotional collapse under the demands of a ruthless instructor.",
			},
			Tags: []string{"drama", "music", "ambition", "obsession"},
		},
		{
			ID:    "movie:la-la-land",
			Label: "La La Land",
			Fields: map[string]any{
				"title": "La La Land",
				"year":  2016,
				"genre": []string{"Musical", "Romance"},
				"cast":  []string{"Ryan Gosling", "Emma Stone"},
				"plot":  "A jazz pianist and an aspiring actress fall in love in Los Angeles, even as their pursuit of their dreams threatens to pull them apart.",
			},
			Tags: []string{"musical", "romance", "melancholic", "dreams", "ambition"},
		},
		{
			ID:    "movie:arrival",
			Label: "Arrival",
			Fields: map[string]any{
				"title": "Arrival",
				"year":  2016,
				"genre": []string{"Sci-Fi", "Drama"},
				"cast":  []string{"Amy Adams", "Jeremy Renner"},
				"plot":  "A linguist is recruited to communicate with alien visitors whose language reshapes her perception of time, memory, and the choices that define a life.",
			},
			Tags: []string{"sci-fi", "drama", "time", "memory", "mind-bending"},
		},
		{
			ID:    "movie:her",
			Label: "Her",
			Fields: map[string]any{
				"title": "Her",
				"year":  2013,
				"genre": []string{"Sci-Fi", "Romance"},
				"cast":  []string{"Joaquin Phoenix", "Scarlett Johansson"},
				"plot":  "In a near future, a lonely writer develops a relationship with an artificially intelligent operating system designed to meet his every need, exploring loneliness, connection, and identity.",
			},
			Tags: []string{"sci-fi", "romance", "loneliness", "identity", "melancholic"},
		},
		{
			ID:    "movie:the-truman-show",
			Label: "The Truman Show",
			Fields: map[string]any{
				"title": "The Truman Show",
				"year":  1998,
				"genre": []string{"Drama", "Sci-Fi"},
				"cast":  []string{"Jim Carrey", "Ed Harris"},
				"plot":  "An insurance salesman gradually discovers that his entire life has been broadcast as a reality television show, and that everyone around him is an actor in a constructed world.",
			},
			Tags: []string{"drama", "sci-fi", "simulated-reality", "identity"},
		},
		{
			ID:    "movie:stand-by-me",
			Label: "Stand By Me",
			Fields: map[string]any{
				"title": "Stand By Me",
				"year":  1986,
				"genre": []string{"Drama", "Adventure"},
				"cast":  []string{"Wil Wheaton", "River Phoenix", "Corey Feldman"},
				"plot":  "Four young friends set out on a journey through the woods to find the body of a missing boy, sharing a final adventure before growing apart on the cusp of adolescence.",
			},
			Tags: []string{"drama", "adventure", "coming-of-age", "friendship", "nostalgia"},
		},
		{
			ID:    "movie:the-perks-of-being-a-wallflower",
			Label: "The Perks of Being a Wallflower",
			Fields: map[string]any{
				"title": "The Perks of Being a Wallflower",
				"year":  2012,
				"genre": []string{"Drama"},
				"cast":  []string{"Logan Lerman", "Emma Watson", "Ezra Miller"},
				"plot":  "A shy teenager navigates his freshman year of high school, finding unlikely friends who help him confront painful memories and find his place in the world.",
			},
			Tags: []string{"drama", "coming-of-age", "melancholic", "friendship"},
		},
		{
			ID:    "movie:mad-max-fury-road",
			Label: "Mad Max: Fury Road",
			Fields: map[string]any{
				"title": "Mad Max: Fury Road",
				"year":  2015,
				"genre": []string{"Action", "Sci-Fi"},
				"cast":  []string{"Tom Hardy", "Charlize Theron"},
				"plot":  "In a desert wasteland ravaged by scarcity, a drifter joins a rebel warrior to flee a tyrannical warlord across a relentless, high-speed chase through the dunes.",
			},
			Tags: []string{"action", "sci-fi", "dystopian", "desert", "epic"},
		},
		{
			ID:    "movie:the-dark-knight",
			Label: "The Dark Knight",
			Fields: map[string]any{
				"title": "The Dark Knight",
				"year":  2008,
				"genre": []string{"Action", "Crime"},
				"cast":  []string{"Christian Bale", "Heath Ledger", "Aaron Eckhart"},
				"plot":  "A vigilante billionaire confronts a chaos-driven criminal mastermind who pushes the city's people, and the hero himself, to their moral breaking points.",
			},
			Tags: []string{"action", "crime", "moral-complexity"},
		},
		{
			ID:    "movie:coco",
			Label: "Coco",
			Fields: map[string]any{
				"title": "Coco",
				"year":  2017,
				"genre": []string{"Animation", "Family"},
				"cast":  []string{"Anthony Gonzalez", "Gael Garcia Bernal"},
				"plot":  "A young boy with dreams of becoming a musician is transported to the land of the dead, where he uncovers his family's hidden history and the true meaning of music and memory.",
			},
			Tags: []string{"animation", "family", "memory", "music", "heartfelt"},
		},
		{
			ID:    "movie:a-quiet-place",
			Label: "A Quiet Place",
			Fields: map[string]any{
				"title": "A Quiet Place",
				"year":  2018,
				"genre": []string{"Horror", "Thriller"},
				"cast":  []string{"Emily Blunt", "John Krasinski"},
				"plot":  "A family struggles to survive in a world hunted by blind creatures with ultra-sensitive hearing, forced to live in near-total silence to avoid being killed.",
			},
			Tags: []string{"horror", "thriller", "survival", "dystopian"},
		},
		{
			ID:    "movie:memento",
			Label: "Memento",
			Fields: map[string]any{
				"title": "Memento",
				"year":  2000,
				"genre": []string{"Thriller", "Mystery"},
				"cast":  []string{"Guy Pearce", "Carrie-Anne Moss"},
				"plot":  "A man with no short-term memory pieces together fragments of his past using notes and tattoos to hunt down the person he believes murdered his wife.",
			},
			Tags: []string{"thriller", "mystery", "memory", "identity", "mind-bending"},
		},
	}
}

// joinOr joins items with ", ", returning fallback if items is empty.
func joinOr(items []string, fallback string) string {
	if len(items) == 0 {
		return fallback
	}
	out := items[0]
	for _, s := range items[1:] {
		out += ", " + s
	}
	return out
}
