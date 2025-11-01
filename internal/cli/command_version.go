package cli

import (
	"fmt"
	"math/rand"
	"os"
	"time"
)

type VersionCommand struct{}

func (_ *VersionCommand) Execute(_ []string) error {
	rng := rand.New(rand.NewSource(time.Now().UnixMilli()))
	randomAnimal := func() uint32 {
		lb, ub := uint32(0x1f400), uint32(0x1f43c)
		return (rng.Uint32() % (ub - lb)) + lb
	}
	fmt.Printf(
		"v(%c).(%c).(%c)\n",
		randomAnimal(), randomAnimal(), randomAnimal(),
	)
	fmt.Fprintln(os.Stderr, "(not versioned, enjoy the animals)")
	return nil
}
