package server

import (
	"archive/zip"
	"bufio"
	"bytes"
	"io"
	"path"
	"regexp"
	"strings"
)

// The built-in detectors: line-oriented regular expressions over the source
// files of an uploaded zip. Scanning is bounded by cgsMaxScanFiles,
// cgsMaxFileBytes, cgsMaxScanBytes and cgsMaxMatches, so its cost doesn't
// grow with the archive.

const (
	cgsMaxUpload     = 4 << 20   // bytes of one uploaded zip
	cgsMaxZipEntries = 2000      // zip entries looked at
	cgsMaxScanFiles  = 200       // source files read
	cgsMaxFileBytes  = 256 << 10 // bytes read per file
	cgsMaxScanBytes  = 2 << 20   // bytes read per archive
	cgsMaxLine       = 1000      // characters matched and kept per line
	cgsMaxMatches    = 500       // findings per archive
	cgsMaxPath       = 512
)

type cgsRule struct {
	id, detector, name, title, description string
	severity, category                     string
	security                               bool // reported for analysisType Security as well as All
	cwe                                    []string
	recommendation, url                    string
	re                                     *regexp.Regexp
	exts                                   string // space-separated extensions; empty: every source file
}

var cgsSourceExts = map[string]bool{
	".py": true, ".js": true, ".jsx": true, ".ts": true, ".tsx": true, ".java": true, ".go": true, ".rb": true,
	".cs": true, ".php": true, ".kt": true, ".scala": true, ".tf": true, ".yaml": true, ".yml": true, ".json": true,
	".sh": true, ".cfg": true, ".ini": true, ".properties": true, ".env": true,
}

var cgsRules = []cgsRule{
	{
		id: "aws-hardcoded-access-key", detector: "hardcoded-credentials", name: "Hardcoded credentials",
		title: "AWS access key ID in source", description: "An AWS access key ID is embedded in the source code.",
		severity: "Critical", category: "Secrets", security: true, cwe: []string{"CWE-798"},
		recommendation: "Remove the key, rotate it, and load credentials from the environment or a secrets manager.",
		url:            "https://cwe.mitre.org/data/definitions/798.html", re: regexp.MustCompile(`\b(AKIA|ASIA)[0-9A-Z]{16}\b`),
	},
	{
		id: "hardcoded-password", detector: "hardcoded-credentials", name: "Hardcoded credentials",
		title: "Password or secret assigned a literal", description: "A password, secret or API key is assigned a string literal.",
		severity: "High", category: "Secrets", security: true, cwe: []string{"CWE-798", "CWE-259"},
		recommendation: "Read secrets from AWS Secrets Manager or the environment instead of source code.",
		url:            "https://cwe.mitre.org/data/definitions/259.html",
		re:             regexp.MustCompile(`(?i)(password|passwd|pwd|secret|api_?key|token)\w*["']?\s*[:=]\s*["'][^"'\s]{4,}["']`),
	},
	{
		id: "code-injection-eval", detector: "code-injection", name: "Code injection",
		title: "Dynamic code evaluation", description: "eval runs code built at run time, which may include untrusted input.",
		severity: "High", category: "Injection", security: true, cwe: []string{"CWE-94", "CWE-95"},
		recommendation: "Parse data with a safe parser (json.loads, ast.literal_eval, JSON.parse) instead of eval.",
		url:            "https://cwe.mitre.org/data/definitions/94.html", re: regexp.MustCompile(`(^|[^\w.])eval\s*\(`),
		exts: ".py .js .jsx .ts .tsx .rb .php",
	},
	{
		id: "os-command-injection", detector: "os-command-injection", name: "OS command injection",
		title: "Shell command built from program data", description: "A shell runs a command line that may include untrusted input.",
		severity: "High", category: "Injection", security: true, cwe: []string{"CWE-77", "CWE-78"},
		recommendation: "Pass an argument list without a shell (subprocess.run([...]), execFile) and validate inputs.",
		url:            "https://cwe.mitre.org/data/definitions/78.html",
		re:             regexp.MustCompile(`os\.system\s*\(|shell\s*=\s*True|child_process\.exec(Sync)?\s*\(|Runtime\.getRuntime\(\)\.exec\s*\(|exec\.Command\(\s*"(sh|bash)"`),
	},
	{
		id: "insecure-tls-verification", detector: "improper-certificate-validation", name: "Improper certificate validation",
		title: "TLS certificate verification disabled", description: "TLS connections accept any certificate, allowing interception.",
		severity: "High", category: "Cryptography", security: true, cwe: []string{"CWE-295"},
		recommendation: "Keep certificate verification on; trust a private CA through the client's CA bundle instead.",
		url:            "https://cwe.mitre.org/data/definitions/295.html",
		re:             regexp.MustCompile(`verify\s*=\s*False|InsecureSkipVerify\s*:\s*true|rejectUnauthorized\s*:\s*false|CERT_NONE`),
	},
	{
		id: "weak-hash-algorithm", detector: "insecure-hashing", name: "Insecure hashing",
		title: "Weak hash algorithm (MD5 or SHA-1)", description: "MD5 and SHA-1 are broken for collision resistance.",
		severity: "Medium", category: "Cryptography", security: true, cwe: []string{"CWE-327", "CWE-328"},
		recommendation: "Use SHA-256 or stronger; for passwords use a slow KDF such as scrypt, bcrypt or Argon2.",
		url:            "https://cwe.mitre.org/data/definitions/328.html",
		re:             regexp.MustCompile(`(?i)hashlib\.(md5|sha1)\b|MessageDigest\.getInstance\(\s*"(MD5|SHA-?1)"|createHash\(\s*['"](md5|sha1)['"]|"crypto/(md5|sha1)"|\b(md5|sha1)\.(New|Sum)\(`),
	},
	{
		id: "cleartext-http-url", detector: "insecure-connection", name: "Insecure connection",
		title: "Cleartext HTTP URL", description: "Data sent over plain HTTP can be read or changed in transit.",
		severity: "Low", category: "Transport", security: true, cwe: []string{"CWE-319"},
		recommendation: "Use HTTPS endpoints.", url: "https://cwe.mitre.org/data/definitions/319.html",
		re: regexp.MustCompile(`["']http://[A-Za-z0-9][^"'\s]*["']`),
	},
	{
		id: "unresolved-todo", detector: "code-quality-todo", name: "Code quality",
		title: "Unresolved TODO or FIXME", description: "A TODO or FIXME marks unfinished code.",
		severity: "Info", category: "Code quality", cwe: []string{},
		recommendation: "Resolve the note or track it in an issue.", url: "https://docs.aws.amazon.com/codeguru/",
		re: regexp.MustCompile(`\b(TODO|FIXME)\b`),
	},
}

type cgsLine struct {
	n    int
	text string
}

type cgsMatch struct {
	rule    int
	path    string
	line    int
	snippet []cgsLine // the line before, the line, the line after
}

func (r *cgsRule) applies(ext string) bool {
	return r.exts == "" || strings.Contains(" "+r.exts+" ", " "+ext+" ")
}

// cgsScanZip runs the rules over the source files in a zip archive and
// returns the matches (at most cgsMaxMatches) and the files read.
func cgsScanZip(data []byte) ([]cgsMatch, int, error) {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, 0, err
	}
	var matches []cgsMatch
	files, budget := 0, cgsMaxScanBytes
	for i, f := range zr.File {
		if i >= cgsMaxZipEntries || files >= cgsMaxScanFiles || budget <= 0 || len(matches) >= cgsMaxMatches {
			break
		}
		ext := strings.ToLower(path.Ext(f.Name))
		if f.FileInfo().IsDir() || len(f.Name) > cgsMaxPath || !cgsSourceExts[ext] {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			continue
		}
		limit := min(cgsMaxFileBytes, budget)
		content, err := io.ReadAll(io.LimitReader(rc, int64(limit)))
		_ = rc.Close()
		if err != nil && len(content) == 0 {
			continue
		}
		files++
		budget -= len(content)
		matches = cgsScanFile(f.Name, ext, content, matches)
	}
	return matches, files, nil
}

// cgsScanFile appends the rule matches in one file's content to matches.
func cgsScanFile(name, ext string, content []byte, matches []cgsMatch) []cgsMatch {
	var lines []string
	sc := bufio.NewScanner(bytes.NewReader(content))
	sc.Buffer(make([]byte, 0, 64<<10), cgsMaxFileBytes+1)
	for sc.Scan() {
		line := sc.Text()
		if len(line) > cgsMaxLine {
			line = line[:cgsMaxLine]
		}
		lines = append(lines, line)
	}
	for i, line := range lines {
		for ri := range cgsRules {
			r := &cgsRules[ri]
			if !r.applies(ext) || !r.re.MatchString(line) {
				continue
			}
			if len(matches) >= cgsMaxMatches {
				return matches
			}
			var snippet []cgsLine
			for j := max(i-1, 0); j <= min(i+1, len(lines)-1); j++ {
				snippet = append(snippet, cgsLine{n: j + 1, text: lines[j]})
			}
			matches = append(matches, cgsMatch{rule: ri, path: name, line: i + 1, snippet: snippet})
		}
	}
	return matches
}
