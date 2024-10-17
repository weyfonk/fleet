package git_test

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/rand"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/rancher/fleet/pkg/git"
)

func getRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	seededRand := rand.New(rand.NewSource(time.Now().UnixNano()))
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}

var _ = Describe("git's validate tests", func() {
	It("Returns no error when branch is correct", func() {
		Expect(git.ValidateBranch("valid")).ToNot(HaveOccurred())
	})

	It("Returns an error when branch is too long", func() {
		longBranch := getRandomString(300)
		err := git.ValidateBranch(longBranch)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(Equal("invalid branch name: too long"))
	})

	It("Returns an error when branch has .lock suffix", func() {
		err := git.ValidateBranch("wrongsuffix.lock")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(Equal("invalid branch name: cannot end with \".lock\""))
	})

	It("Returns an error when branch starts with .", func() {
		err := git.ValidateBranch(".wrongprefix")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(Equal("invalid branch name: cannot start with \".\""))
	})

	It("Returns an error when branch equals @", func() {
		err := git.ValidateBranch("@")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(Equal("invalid branch name: \"@\""))
	})

	It("Returns an error when branch contains non supported substring", func() {
		var branchInvalidContains = []string{
			"..",
			"//",
			"?",
			"*",
			"[",
			"@{",
			"\\",
			" ",
			"~",
			"^",
			":",
		}
		for _, substr := range branchInvalidContains {
			branch := "test" + substr + "branch"
			err := git.ValidateBranch(branch)
			Expect(err).To(HaveOccurred())
			message := fmt.Sprintf("invalid branch name: cannot contain %q", substr)
			Expect(err.Error()).To(Equal(message))
		}
	})

	It("Returns an error when branch contains a unicode control char", func() {
		const controlChars = "\x00\x01\x02\x03\x04\x05\x06\x07\x08\x09\x0a\x0b\x0c\x0d\x0e\x0f\x7f"
		branch := getRandomString(10)
		byteBranch := []byte(branch)
		for _, c := range controlChars {
			byteBranch[0] = byte(c)
			err := git.ValidateBranch(string(byteBranch))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("invalid branch name: control chars are not supported"))
		}
	})

	It("Returns no error when commit is a valid sha1", func() {
		commit := sha1.Sum([]byte(getRandomString(10)))
		commitStr := hex.EncodeToString(commit[:])
		err := git.ValidateCommit(string(commitStr))
		Expect(err).ToNot(HaveOccurred())
	})

	It("Returns no error when commit is a valid sha256", func() {
		commit := sha256.Sum256([]byte(getRandomString(10)))
		commitStr := hex.EncodeToString(commit[:])
		err := git.ValidateCommit(string(commitStr))
		Expect(err).ToNot(HaveOccurred())
	})

	It("Returns error when commit is not a valid sha256 not sha1", func() {
		commit := md5.Sum([]byte(getRandomString(10)))
		commitStr := hex.EncodeToString(commit[:])
		err := git.ValidateCommit(string(commitStr))
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(Equal(fmt.Sprintf("invalid commit ID: %q", commitStr)))
	})

	It("Returns error when commit has the right length but is not a valid digest", func() {
		commit := getRandomString(40)
		err := git.ValidateCommit(commit)
		Expect(err).To(HaveOccurred())
		Expect(err).To(Equal(fmt.Errorf("invalid commit ID: %q is not a valid hex", commit)))
	})
})
