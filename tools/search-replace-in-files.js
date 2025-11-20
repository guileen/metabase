#!/usr/bin/env node

const fs = require('fs');
const path = require('path');
const readline = require('readline');

/**
 * Search and Replace in Files Tool
 *
 * A powerful tool designed for AI-assisted programming that allows batch search/replace
 * operations with preview capabilities to reduce context waste and improve efficiency.
 *
 * Usage: node search-replace-in-files.js [options] <search> <replace> [files...]
 *
 * Options:
 *   --regex, -r           Use regex search (default: string search)
 *   --preview, -p         Show preview only (don't replace)
 *   --force, -f           Force replace all without confirmation
 *   --context, -C <num>   Show N lines of context (default: 3)
 *   --after, -A <num>     Show N lines after (default: 3)
 *   --before, -B <num>    Show N lines before (default: 3)
 *   --ignore-case, -i     Case insensitive search (regex only)
 *   --glob, -g <pattern>  File pattern (default: \"*.*\")
 *   --confirm, -c         Interactive confirmation by replacement number
 *   --help, -h            Show this help
 *
 * Examples:
 *   node search-replace-in-files.js --preview "oldFunction" "newFunction" \"*.go\"
 *   node search-replace-in-files.js --regex --preview "func (\\\\w+)\\\\(" "func $1_new(" \"*.js\"
 *   node search-replace-in-files.js --confirm "import.*old" "import new" \"*.ts\"
 *   node search-replace-in-files.js --force --regex "TODO:" "FIXME:" \"*.*\"
 */

class SearchReplaceTool {
  constructor() {
    this.options = {
      regex: false,
      preview: false,
      force: false,
      context: 3,
      after: 3,
      before: 3,
      ignoreCase: false,
      glob: '**/*',
      apply: null,
      ai: false,
      searchPattern: null,
      replacement: null,
      files: []
    };

    this.matches = [];
  }

  /**
   * Print help information
   */
  showHelp() {
    console.log(`
Search and Replace in Files Tool

A powerful tool designed for AI-assisted programming that allows batch search/replace
operations with preview capabilities to reduce context waste and improve efficiency.

Usage: node search-replace-in-files.js [options] <search> <replace> [files...]

Options:
  --regex, -r           Use regex search (default: string search)
  --preview, -p         Show preview only (don't replace)
  --force, -f           Force replace all without confirmation
  --context, -C <num>   Show N lines of context (default: 3)
  --after, -A <num>     Show N lines after (default: 3)
  --before, -B <num>    Show N lines before (default: 3)
  --ignore-case, -i     Case insensitive search (regex only)
  --glob, -g <pattern>  File pattern (default: **/*)
  --confirm, -c         Interactive confirmation by replacement number
  --help, -h            Show this help

Examples:
  node search-replace-in-files.js --preview "oldFunction" "newFunction" "**/*.go"
  node search-replace-in-files.js --regex --preview "func (\\\\w+)\\\\(" "func $1_new(" "**/*.js"
  node search-replace-in-files.js --confirm "import.*old" "import new" "**/*.ts"
  node search-replace-in-files.js --force --regex "TODO:" "FIXME:" "**/*"

AI Usage Pattern:
1. Use --preview first to see all potential changes
2. Use --confirm to selectively apply changes
3. Use --force for bulk operations after preview confirmation
`);
  }

  /**
   * Parse command line arguments
   */
  parseArgs(args) {
    const remainingArgs = [];

    for (let i = 0; i < args.length; i++) {
      const arg = args[i];

      switch (arg) {
        case '--help':
        case '-h':
          this.showHelp();
          process.exit(0);

        case '--regex':
        case '-r':
          this.options.regex = true;
          break;

        case '--preview':
        case '-p':
          this.options.preview = true;
          break;

        case '--force':
        case '-f':
          this.options.force = true;
          break;

        // confirm option is removed as interactive mode is disabled

        case '--apply':
        case '-a':
          this.options.apply = args[++i];
          break;

        case '--ai':
        case '-ai':
          this.options.ai = true;
          break;

        case '--ignore-case':
        case '-i':
          this.options.ignoreCase = true;
          break;

        case '--context':
        case '-C':
          this.options.context = parseInt(args[++i]) || 3;
          this.options.before = this.options.context;
          this.options.after = this.options.context;
          break;

        case '--after':
        case '-A':
          this.options.after = parseInt(args[++i]) || 3;
          break;

        case '--before':
        case '-B':
          this.options.before = parseInt(args[++i]) || 3;
          break;

        case '--glob':
        case '-g':
          this.options.glob = args[++i];
          break;

        default:
          if (!arg.startsWith('-')) {
            remainingArgs.push(arg);
          } else {
            console.error(`Unknown option: ${arg}`);
            process.exit(1);
          }
      }
    }

    if (remainingArgs.length < 2) {
      console.error('Error: Search pattern and replacement are required');
      console.error('Use --help for usage information');
      process.exit(1);
    }

    this.options.searchPattern = remainingArgs[0];
    this.options.replacement = remainingArgs[1];
    this.options.files = remainingArgs.slice(2);

    if (this.options.files.length === 0) {
      this.options.files = [this.options.glob];
    }
  }

  /**
   * Convert glob pattern to regex for file matching
   */
  globToRegex(glob) {
    // Convert glob pattern to regex
    let regexPattern = glob
      .replace(/[.*+?^${}()|[\]\\]/g, '\\$&') // Escape regex special chars
      .replace(/\\\*/g, '.*') // Convert * to .*
      .replace(/\\\?/g, '.');  // Convert ? to .

    return new RegExp('^' + regexPattern + '$');
  }

  /**
   * Find files matching the patterns
   */
  findFiles(patterns) {
    const files = new Set();
    const cwd = process.cwd();

    const self = this;
    function scanDirectory(dir) {
      try {
        const items = fs.readdirSync(dir);

        for (const item of items) {
          const fullPath = path.join(dir, item);
          const stat = fs.statSync(fullPath);
          const relativePath = path.relative(cwd, fullPath);

          if (stat.isDirectory()) {
            // Skip .git, node_modules, vendor directories
            if (item !== '.git' && item !== 'node_modules' && item !== 'vendor' && item !== '.next' && item !== 'dist') {
              scanDirectory(fullPath);
            }
          } else {
            // Check if file matches any pattern
            for (const pattern of patterns) {
              const regex = self.globToRegex(pattern);
              if (regex.test(relativePath)) {
                // Only include text files
                if (self.isTextFile(fullPath)) {
                  files.add(relativePath);
                }
                break;
              }
            }
          }
        }
      } catch (error) {
        // Skip directories we can't read
      }
    }

    scanDirectory(cwd);
    return Array.from(files);
  }

  /**
   * Check if file is a text file and filter out minified/obfuscated files
   */
  isTextFile(filePath) {
    try {
      // Skip the tool itself
      if (filePath.includes('search-replace-in-files.js')) {
        return false;
      }

      const buffer = fs.readFileSync(filePath, { encoding: null });
      const content = buffer.toString('utf8');

      // Check for binary indicators
      for (let i = 0; i < Math.min(buffer.length, 1024); i++) {
        const byte = buffer[i];
        if (byte === 0) return false; // Null byte indicates binary
        if (byte < 32 && byte !== 9 && byte !== 10 && byte !== 13) return false; // Control characters
      }

      // Check if it looks like minified/obfuscated JS
      if (filePath.endsWith('.js')) {
        const lines = content.split('\n');
        const avgLineLength = lines.reduce((sum, line) => sum + line.length, 0) / lines.length;
        const noSpacesRatio = content.replace(/\s/g, '').length / content.length;
        const suspiciousChars = /[a-z]{30,}/.test(content) || /,[a-z]{2,},/.test(content) || /function\([a-z],[a-z]\)/.test(content);

        // More liberal filtering - only filter extremely minified files
        if (avgLineLength > 500 || (noSpacesRatio > 0.85 && avgLineLength > 100) || suspiciousChars) {
          return false; // Likely minified/obfuscated
        }
      }

      return true;
    } catch (error) {
      return false;
    }
  }

  /**
   * Search for pattern in a file
   */
  searchInFile(filePath) {
    try {
      const content = fs.readFileSync(filePath, 'utf8');
      const lines = content.split('\n');
      const fileMatches = [];

      let searchRegex;
      if (this.options.regex) {
        const flags = this.options.ignoreCase ? 'gi' : 'g';
        searchRegex = new RegExp(this.options.searchPattern, flags);
      } else {
        // String search - create regex for exact match
        const escapedPattern = this.options.searchPattern.replace(/[.*+?^${}()|[\]/\\]/g, '\\$&');
        const flags = this.options.ignoreCase ? 'gi' : 'g';
        searchRegex = new RegExp(escapedPattern, flags);
      }

      lines.forEach((line, lineIndex) => {
        const matches = [...line.matchAll(searchRegex)];
        matches.forEach(match => {
          fileMatches.push({
            lineNumber: lineIndex + 1,
            line: line,
            match: match[0],
            start: match.index,
            end: match.index + match[0].length,
            groups: match.slice(1)
          });
        });
      });

      return fileMatches;
    } catch (error) {
      console.error(`Error reading ${filePath}: ${error.message}`);
      return [];
    }
  }

  /**
   * Create diff-style preview
   */
  createPreview(filePath, matches) {
    if (matches.length === 0) return '';

    try {
      const content = fs.readFileSync(filePath, 'utf8');
      const lines = content.split('\n');
      const relativePath = path.relative(process.cwd(), filePath);
      let preview = '';

      if (this.options.ai) {
        // AI-friendly markdown format - similar to git diff
        preview += `\n## ${relativePath}\n\n\`\`\`diff\n`;
      } else {
        preview += `\n📁 File: ${relativePath}\n`;
        preview += `${''.padEnd(40, '-')}\n`;
      }

      let matchIndex = 1;

      for (const match of matches) {
        const lineNum = match.lineNumber;
        const startLine = Math.max(1, lineNum - this.options.before);
        const endLine = Math.min(lines.length, lineNum + this.options.after);

        // Show context lines
        for (let i = startLine; i <= endLine; i++) {
          let displayLine = lines[i - 1];

          if (i === lineNum) {
            // Show the match with diff format
            const before = displayLine.substring(0, match.start);
            const matched = displayLine.substring(match.start, match.end);
            const after = displayLine.substring(match.end);

            let replacement = this.options.replacement;
            if (this.options.regex && match.groups.length > 0) {
              match.groups.forEach((group, index) => {
                replacement = replacement.replace(new RegExp(`\\$${index + 1}`, 'g'), group || '');
              });
            }

            if (this.options.ai) {
              // AI format: proper diff format
              preview += `@@ -${lineNum} +${lineNum} @@\n`;
              preview += `- ${before}${matched}${after}\n`;
              preview += `+ ${before}${replacement}${after}\n`;
            } else {
              // Original format with colors
              const lineNumber = String(i).padStart(4, ' ');
              preview += `${lineNumber}: ${before}\x1b[31m${matched}\x1b[0m${after}\n`;
              preview += `${' '.repeat(5)}+ ${before}\x1b[32m${replacement}\x1b[0m${after}\n`;
            }
          } else {
            if (this.options.ai) {
              // Context lines in diff format
              preview += ` ${displayLine}\n`;
            } else {
              const prefix = ' ';
              const lineNumber = String(i).padStart(4, ' ');
              preview += `${prefix}${lineNumber}: ${displayLine}\n`;
            }
          }
        }

        matchIndex++;
      }

      if (this.options.ai) {
        preview += '```\n';
      }

      return preview;
    } catch (error) {
      return this.options.ai ?
        `Error: ${filePath}: ${error.message}` :
        `Error creating preview for ${filePath}: ${error.message}`;
    }
  }

  /**
   * Perform replacements in a file
   */
  replaceInFile(filePath, selectedMatches = null) {
    try {
      let content = fs.readFileSync(filePath, 'utf8');
      let modified = false;
      let replacementsMade = 0;

      if (selectedMatches === null) {
        // Replace all matches
        let searchRegex;
        if (this.options.regex) {
          const flags = this.options.ignoreCase ? 'gi' : 'g';
          searchRegex = new RegExp(this.options.searchPattern, flags);
        } else {
          const escapedPattern = this.options.searchPattern.replace(/[.*+?^${}()|[\]/\\]/g, '\\$&');
          const flags = this.options.ignoreCase ? 'gi' : 'g';
          searchRegex = new RegExp(escapedPattern, flags);
        }

        const originalContent = content;
        content = content.replace(searchRegex, (match, ...args) => {
          let replacement = this.options.replacement;
          if (this.options.regex && args.length > 0) {
            // Replace regex groups
            args.slice(0, -2).forEach((group, index) => {
              replacement = replacement.replace(new RegExp(`\\$${index + 1}`, 'g'), group || '');
            });
          }
          replacementsMade++;
          return replacement;
        });

        modified = originalContent !== content;
      } else {
        // Replace only selected matches by line number
        const lines = content.split('\n');
        const fileMatches = this.searchInFile(filePath);

        selectedMatches.forEach(matchIndex => {
          if (matchIndex >= 1 && matchIndex <= fileMatches.length) {
            const match = fileMatches[matchIndex - 1];
            const lineNum = match.lineNumber - 1; // Convert to 0-based
            let line = lines[lineNum];

            // Perform replacement on the specific line
            let replacement = this.options.replacement;
            if (this.options.regex && match.groups.length > 0) {
              match.groups.forEach((group, index) => {
                replacement = replacement.replace(new RegExp(`\\$${index + 1}`, 'g'), group || '');
              });
            }

            lines[lineNum] = line.substring(0, match.start) + replacement + line.substring(match.end);
            replacementsMade++;
            modified = true;
          }
        });

        content = lines.join('\n');
      }

      if (modified) {
        fs.writeFileSync(filePath, content, 'utf8');
        return replacementsMade;
      }

      return 0;
    } catch (error) {
      console.error(`Error replacing in ${filePath}: ${error.message}`);
      return 0;
    }
  }

  // Interactive mode is disabled, so promptSelection method is removed

  /**
   * Parse user selection input
   */
  parseSelections(input, totalMatches) {
    const selections = new Set();

    if (!input || input.toLowerCase() === 'none') {
      return selections;
    }

    if (input.toLowerCase() === 'all') {
      for (let i = 1; i <= totalMatches; i++) {
        selections.add(i);
      }
      return selections;
    }

    // Parse ranges and individual numbers (e.g., "1,3,5-8,12")
    const parts = input.split(',');
    for (const part of parts) {
      const trimmed = part.trim();
      if (trimmed.includes('-')) {
        const [start, end] = trimmed.split('-').map(n => parseInt(n.trim()));
        if (!isNaN(start) && !isNaN(end)) {
          for (let i = Math.max(1, start); i <= Math.min(totalMatches, end); i++) {
            selections.add(i);
          }
        }
      } else {
        const num = parseInt(trimmed);
        if (!isNaN(num) && num >= 1 && num <= totalMatches) {
          selections.add(num);
        }
      }
    }

    return selections;
  }

  /**
   * Main execution function
   */
  async run() {
    console.log('🔍 Search and Replace Tool\n');

    const files = this.findFiles(this.options.files);
    console.log(`📁 Scanning ${files.length} files...`);

    let totalMatches = 0;

    // Search for matches in all files
    for (const filePath of files) {
      const matches = this.searchInFile(filePath);
      if (matches.length > 0) {
        this.matches.push({
          filePath,
          matches,
          matchCount: matches.length
        });
        totalMatches += matches.length;
      }
    }

    if (this.options.ai) {
      console.log(`Matches: ${totalMatches} in ${this.matches.length} files`);
    } else {
      console.log(`\n📊 Found ${totalMatches} matches in ${this.matches.length} files.`);
    }

    if (totalMatches === 0) {
      console.log(this.options.ai ? 'No matches found.' : '✅ No matches found. Nothing to replace.');
      return;
    }

    let globalMatchIndex = 1;
    for (const fileMatch of this.matches) {
      const preview = this.createPreview(fileMatch.filePath, fileMatch.matches);
      if (preview) {
        console.log(preview);
        globalMatchIndex += fileMatch.matchCount;
      }
    }

    if (this.options.preview) {
      if (this.options.ai) {
        console.log(`\n## Summary\n\nFound ${totalMatches} matches in ${this.matches.length} files.\n\nUsage: node search-replace-in-files.js --ai --apply "1,3,5"`);
      } else {
        console.log('\n👀 Preview mode - no files were modified.');
        console.log('\n💡 To apply these replacements, run:');
        console.log(`   node search-replace-in-files.js --force "${this.options.searchPattern}" "${this.options.replacement}" "${this.options.files[0]}"`);
      }
      return;
    }

    if (this.options.apply) {
      // Apply specific replacements
      const selectedMatches = this.parseSelections(this.options.apply, totalMatches);

      if (selectedMatches.size === 0) {
        console.log(this.options.ai ? 'No replacements selected.' : '❌ No replacements selected.');
        return;
      }

      console.log(this.options.ai ?
        `Applying ${selectedMatches.size} replacements...` :
        `\n🎯 Applying ${selectedMatches.size} selected replacements...`);

      let currentMatchIndex = 1;
      let totalReplacements = 0;

      for (const fileMatch of this.matches) {
        const fileStartIndex = currentMatchIndex;
        const fileEndIndex = currentMatchIndex + fileMatch.matchCount - 1;

        // Find which selected matches belong to this file
        const fileSelections = [];
        for (let i = fileStartIndex; i <= fileEndIndex; i++) {
          if (selectedMatches.has(i)) {
            fileSelections.push(i - fileStartIndex + 1); // Convert to local file index
          }
        }

        if (fileSelections.length > 0) {
          const replacements = this.replaceInFile(fileMatch.filePath, fileSelections);
          totalReplacements += replacements;
          if (this.options.ai) {
            console.log(`${fileMatch.filePath}: ${replacements} replacements`);
          } else {
            console.log(`✅ ${fileMatch.filePath}: ${replacements} replacements (matches: ${fileSelections.join(',')})`);
          }
        }

        currentMatchIndex = fileEndIndex + 1;
      }

      console.log(this.options.ai ?
        `Complete! ${totalReplacements} replacements made.` :
        `\n🎉 Complete! ${totalReplacements} replacements made.`);
      return;
    }

    if (this.options.force) {
      // Replace all matches
      console.log(this.options.ai ? 'Applying all replacements...' : '\n⚡ Force mode - applying all replacements...');
      let totalReplacements = 0;

      for (const fileMatch of this.matches) {
        const replacements = this.replaceInFile(fileMatch.filePath);
        totalReplacements += replacements;
        if (replacements > 0) {
          if (this.options.ai) {
            console.log(`${fileMatch.filePath}: ${replacements} replacements`);
          } else {
            console.log(`✅ ${fileMatch.filePath}: ${replacements} replacements`);
          }
        }
      }

      console.log(this.options.ai ?
        `Complete! ${totalReplacements} replacements made across ${this.matches.length} files.` :
        `\n🎉 Complete! ${totalReplacements} replacements made across ${this.matches.length} files.`);
      return;
    }

    // Default behavior - show preview and exit if no action is specified
    console.log(this.options.ai ? '\nPreview completed.' : '\n👋 Goodbye!');
  }
}

// Main execution
async function main() {
  const args = process.argv.slice(2);
  const tool = new SearchReplaceTool();

  try {
    tool.parseArgs(args);
    await tool.run();
  } catch (error) {
    console.error('Error:', error.message);
    process.exit(1);
  }
}

if (require.main === module) {
  main();
}

module.exports = SearchReplaceTool;