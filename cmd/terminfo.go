package main

import (
	"fmt"
	"os"
	"unicode/utf8"

	"github.com/gdamore/tcell/v2"
	"github.com/mattn/go-runewidth"
)

func main() {
	// Affiche quelques infos d'environnement
	fmt.Println("TERM:", os.Getenv("TERM"))
	fmt.Println("LANG:", os.Getenv("LANG"))

	// Exemple de caractères à tester
	samples := []string{
		"A", // simple ASCII
		"é", // accentué
		"─", // box-drawing
		"𞠻", // caractère
		"😊", // emoji
	}

	fmt.Println("\n== Largeur calculée par Go (utf8.RuneLen + rune width) ==")
	for _, s := range samples {
		r, _ := utf8.DecodeRuneInString(s)
		fmt.Printf("%q: rune=%U, utf8len=%d\n", s, r, utf8.RuneLen(r))
	}

	// Maintenant on demande à tcell
	screen, err := tcell.NewScreen()
	if err != nil {
		fmt.Println("Erreur init tcell:", err)
		return
	}
	defer func() {
		if screen != nil {
			screen.Fini()
		}
	}()

	fmt.Println("\n== Largeur selon tcell ==")
	for _, s := range samples {
		r, _ := utf8.DecodeRuneInString(s)
		w := runewidth.StringWidth(string(r))
		fmt.Printf("%q: width=%d\n", s, w)
	}

}
