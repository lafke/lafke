package ecosystem

import (
	"github.com/lafke/lafke/internal/ecosystem/bun"
	"github.com/lafke/lafke/internal/ecosystem/bundler"
	"github.com/lafke/lafke/internal/ecosystem/cabal"
	"github.com/lafke/lafke/internal/ecosystem/cargo"
	"github.com/lafke/lafke/internal/ecosystem/cocoapods"
	"github.com/lafke/lafke/internal/ecosystem/composer"
	"github.com/lafke/lafke/internal/ecosystem/conan"
	"github.com/lafke/lafke/internal/ecosystem/deno"
	"github.com/lafke/lafke/internal/ecosystem/golang"
	"github.com/lafke/lafke/internal/ecosystem/gradle"
	"github.com/lafke/lafke/internal/ecosystem/julia"
	"github.com/lafke/lafke/internal/ecosystem/mix"
	"github.com/lafke/lafke/internal/ecosystem/npm"
	"github.com/lafke/lafke/internal/ecosystem/nuget"
	"github.com/lafke/lafke/internal/ecosystem/pdm"
	"github.com/lafke/lafke/internal/ecosystem/pipenv"
	"github.com/lafke/lafke/internal/ecosystem/pnpm"
	"github.com/lafke/lafke/internal/ecosystem/poetry"
	"github.com/lafke/lafke/internal/ecosystem/pub"
	"github.com/lafke/lafke/internal/ecosystem/pypi"
	"github.com/lafke/lafke/internal/ecosystem/renv"
	"github.com/lafke/lafke/internal/ecosystem/swiftpm"
	"github.com/lafke/lafke/internal/ecosystem/yarn"
)

func All() []Ecosystem {
	return []Ecosystem{
		pnpm.New(),
		yarn.New(),
		bun.New(),
		npm.New(),
		deno.New(),
		cargo.New(),
		golang.New(),
		pypi.New(),
		poetry.New(),
		pdm.New(),
		pipenv.New(),
		composer.New(),
		bundler.New(),
		nuget.New(),
		swiftpm.New(),
		cocoapods.New(),
		conan.New(),
		renv.New(),
		julia.New(),
		cabal.New(),
		pub.New(),
		mix.New(),
		gradle.New(),
	}
}

func Prefer(found []Ecosystem) []Ecosystem {
	found = preferFamily(found, []string{"pnpm", "yarn", "bun", "npm"})
	found = preferFamily(found, []string{"pypi", "poetry", "pdm", "pipenv"})
	return found
}

func preferFamily(found []Ecosystem, order []string) []Ecosystem {
	present := map[string]bool{}
	member := map[string]bool{}
	for _, name := range order {
		member[name] = true
	}
	for _, e := range found {
		present[e.Name()] = true
	}
	chosen := ""
	for _, name := range order {
		if present[name] {
			chosen = name
			break
		}
	}
	if chosen == "" {
		return found
	}
	out := make([]Ecosystem, 0, len(found))
	for _, e := range found {
		if member[e.Name()] && e.Name() != chosen {
			continue
		}
		out = append(out, e)
	}
	return out
}
