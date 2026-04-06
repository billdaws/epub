// Creates a semver git tag. If no semver tags exist the first tag defaults
// to v1.0.0. Otherwise the user is prompted to enter a tag; the default is
// the latest tag with the patch version incremented.
//
// Usage: semver
package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
)

type semver struct {
	major, minor, patch int
}

func (s semver) String() string {
	return fmt.Sprintf("v%d.%d.%d", s.major, s.minor, s.patch)
}

func parse(tag string) (semver, bool) {
	s := strings.TrimPrefix(tag, "v")
	parts := strings.Split(s, ".")
	if len(parts) != 3 {
		return semver{}, false
	}
	nums := make([]int, 3)
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 {
			return semver{}, false
		}
		nums[i] = n
	}
	return semver{nums[0], nums[1], nums[2]}, true
}

func latestTag() (semver, bool) {
	out, err := exec.Command("git", "tag", "--list", "v*").Output()
	if err != nil {
		return semver{}, false
	}

	var versions []semver
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if v, ok := parse(line); ok {
			versions = append(versions, v)
		}
	}

	if len(versions) == 0 {
		return semver{}, false
	}

	sort.Slice(versions, func(i, j int) bool {
		a, b := versions[i], versions[j]
		if a.major != b.major {
			return a.major > b.major
		}
		if a.minor != b.minor {
			return a.minor > b.minor
		}
		return a.patch > b.patch
	})
	return versions[0], true
}

func main() {
	latest, exists := latestTag()

	var defaultTag semver
	if !exists {
		defaultTag = semver{1, 0, 0}
	} else {
		defaultTag = semver{latest.major, latest.minor, latest.patch + 1}
	}

	if exists {
		fmt.Printf("latest tag: %s\n", latest)
	} else {
		fmt.Println("no semver tags found")
	}
	fmt.Printf("new tag [%s]: ", defaultTag)

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		fmt.Fprintf(os.Stderr, "semver: %v\n", err)
		os.Exit(1)
	}
	input = strings.TrimSpace(input)

	var chosen semver
	if input == "" {
		chosen = defaultTag
	} else {
		v, ok := parse(input)
		if !ok {
			fmt.Fprintf(os.Stderr, "semver: invalid tag %q — must be vMAJOR.MINOR.PATCH\n", input)
			os.Exit(1)
		}
		chosen = v
	}

	cmd := exec.Command("git", "tag", chosen.String())
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "semver: git tag failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("tagged %s\n", chosen)
}
