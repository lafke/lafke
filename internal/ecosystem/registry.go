package ecosystem

import (
	"github.com/lafke/cli/internal/ecosystem/bun"
	"github.com/lafke/cli/internal/ecosystem/bundler"
	"github.com/lafke/cli/internal/ecosystem/cabal"
	"github.com/lafke/cli/internal/ecosystem/cargo"
	"github.com/lafke/cli/internal/ecosystem/cocoapods"
	"github.com/lafke/cli/internal/ecosystem/composer"
	"github.com/lafke/cli/internal/ecosystem/conan"
	"github.com/lafke/cli/internal/ecosystem/deno"
	"github.com/lafke/cli/internal/ecosystem/golang"
	"github.com/lafke/cli/internal/ecosystem/gradle"
	"github.com/lafke/cli/internal/ecosystem/julia"
	"github.com/lafke/cli/internal/ecosystem/mix"
	"github.com/lafke/cli/internal/ecosystem/npm"
	"github.com/lafke/cli/internal/ecosystem/nuget"
	"github.com/lafke/cli/internal/ecosystem/pdm"
	"github.com/lafke/cli/internal/ecosystem/pipenv"
	"github.com/lafke/cli/internal/ecosystem/pnpm"
	"github.com/lafke/cli/internal/ecosystem/poetry"
	"github.com/lafke/cli/internal/ecosystem/pub"
	"github.com/lafke/cli/internal/ecosystem/pypi"
	"github.com/lafke/cli/internal/ecosystem/renv"
	"github.com/lafke/cli/internal/ecosystem/swiftpm"
	"github.com/lafke/cli/internal/ecosystem/yarn"
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
