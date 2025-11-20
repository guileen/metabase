#!/usr/bin/env node

const fs = require('fs');
const path = require('path');
const os = require('os');

/**
 * Go Module Movement Tool
 *
 * A universal tool for moving Go modules/packages and updating all import references.
 * Supports both file paths and Go package names.
 *
 * Usage: node move-go-module.js <from> <to>
 * Examples:
 *   node move-go-module.js internal/common pkg/common
 *   node move-go-module.js github.com/old/repo/internal/utils github.com/new/repo/pkg/utils
 *   node move-go-module.js ./internal/api ./pkg/api
 */

class GoModuleMover {
  constructor() {
    this.processedFiles = new Set();
    this.errors = [];
    this.movedFiles = [];
  }

  log(level, message) {
    const timestamp = new Date().toISOString().split('T')[1].split('.')[0];
    console.log(`[${timestamp}] ${level}: ${message}`);
  }

  info(message) {
    this.log('INFO', message);
  }

  warn(message) {
    this.log('WARN', message);
  }

  error(message) {
    this.log('ERROR', message);
    this.errors.push(message);
  }

  /**
   * Normalize a path to be absolute and handle different path formats
   */
  normalizePath(inputPath) {
    // Handle Go module paths (like github.com/user/repo/pkg)
    if (inputPath.includes('/') && !inputPath.startsWith('./') && !path.isAbsolute(inputPath)) {
      return inputPath;
    }

    // Handle relative and absolute file system paths
    let resolvedPath = path.resolve(inputPath);
    if (fs.existsSync(resolvedPath)) {
      return resolvedPath;
    }

    // If path doesn't exist, it might be a Go module path that doesn't exist locally yet
    return inputPath;
  }

  /**
   * Check if a path is a Go module path vs file system path
   */
  isGoModulePath(pathStr) {
    return !path.isAbsolute(pathStr) &&
           !pathStr.startsWith('./') &&
           !pathStr.startsWith('../') &&
           pathStr.includes('/') &&
           !fs.existsSync(pathStr);
  }

  /**
   * Extract base name from a path (last component)
   */
  getBaseName(pathStr) {
    return path.basename(pathStr);
  }

  /**
   * Check if directory is not empty
   */
  isDirectoryEmpty(dirPath) {
    if (!fs.existsSync(dirPath)) {
      return true;
    }

    try {
      const files = fs.readdirSync(dirPath);
      return files.length === 0;
    } catch (error) {
      return false;
    }
  }

  /**
   * Find all Go files in the current project recursively
   */
  findGoFiles(rootPath = '.') {
    const goFiles = [];

    function scanDirectory(dir) {
      try {
        const items = fs.readdirSync(dir);

        for (const item of items) {
          const fullPath = path.join(dir, item);
          const stat = fs.statSync(fullPath);

          if (stat.isDirectory()) {
            // Skip vendor and .git directories
            if (item !== 'vendor' && item !== '.git' && item !== 'node_modules') {
              scanDirectory(fullPath);
            }
          } else if (item.endsWith('.go')) {
            goFiles.push(fullPath);
          }
        }
      } catch (error) {
        // Skip directories we can't read
      }
    }

    scanDirectory(rootPath);
    return goFiles;
  }

  /**
   * Read go.mod file to get module name
   */
  getModuleName() {
    const goModPath = path.join(process.cwd(), 'go.mod');

    if (!fs.existsSync(goModPath)) {
      this.warn('go.mod not found in current directory');
      return null;
    }

    try {
      const content = fs.readFileSync(goModPath, 'utf8');
      const match = content.match(/^module\s+([^\s\n]+)/m);
      return match ? match[1] : null;
    } catch (error) {
      this.error(`Failed to read go.mod: ${error.message}`);
      return null;
    }
  }

  /**
   * Convert various import path formats to canonical form
   */
  normalizeImportPath(importPath, moduleName) {
    // Remove leading ./
    if (importPath.startsWith('./')) {
      importPath = importPath.substring(2);
    }

    // If it doesn't start with a known domain, prepend module name
    if (moduleName && !importPath.includes('.') && !importPath.startsWith(moduleName)) {
      importPath = `${moduleName}/${importPath}`;
    }

    return importPath;
  }

  /**
   * Update import statements in a Go file
   */
  updateImportsInFile(filePath, fromPath, toPath, moduleName) {
    try {
      const content = fs.readFileSync(filePath, 'utf8');
      const originalContent = content;

      // Normalize import paths
      const normalizedFrom = this.normalizeImportPath(fromPath, moduleName);
      const normalizedTo = this.normalizeImportPath(toPath, moduleName);

      // Pattern to match import statements (including multi-line imports)
      const importRegex = /import\s*\(\s*([^)]*?)\s*\)/gs;
      const singleImportRegex = /import\s+([^\s\n]+)/g;

      let updatedContent = content;

      // Handle multi-line imports
      updatedContent = updatedContent.replace(importRegex, (match, importsBlock) => {
        const updatedImports = importsBlock.replace(/([^\s\n]+)(\s*\/\/.*)?/g, (importMatch) => {
          const importPath = importMatch.trim().split(/\s+/)[0];
          if (importPath === normalizedFrom || importPath === fromPath) {
            return importMatch.replace(importPath, normalizedTo);
          }
          return importMatch;
        });
        return `import (${updatedImports})`;
      });

      // Handle single line imports
      updatedContent = updatedContent.replace(singleImportRegex, (match, importPath) => {
        if (importPath === normalizedFrom || importPath === fromPath || importPath === `"${normalizedFrom}"` || importPath === `"${fromPath}"`) {
          const cleanPath = importPath.replace(/"/g, '');
          return match.replace(cleanPath, normalizedTo);
        }
        return match;
      });

      // Also handle quoted imports in general text
      const quoteRegex = /"([^"]+)"/g;
      updatedContent = updatedContent.replace(quoteRegex, (match, importPath) => {
        if (importPath === normalizedFrom || importPath === fromPath) {
          return `"${normalizedTo}"`;
        }
        return match;
      });

      if (updatedContent !== originalContent) {
        fs.writeFileSync(filePath, updatedContent, 'utf8');
        this.info(`Updated imports in ${filePath}`);
        return true;
      }

      return false;
    } catch (error) {
      this.error(`Failed to update imports in ${filePath}: ${error.message}`);
      return false;
    }
  }

  /**
   * Move a directory recursively
   */
  moveDirectory(fromPath, toPath) {
    try {
      // Create parent directories if they don't exist
      const parentDir = path.dirname(toPath);
      if (!fs.existsSync(parentDir)) {
        fs.mkdirSync(parentDir, { recursive: true });
      }

      // If destination exists and is not empty, handle it
      if (fs.existsSync(toPath) && !this.isDirectoryEmpty(toPath)) {
        this.warn(`Destination directory ${toPath} is not empty. Files will be merged.`);
      }

      // Ensure destination directory exists
      if (!fs.existsSync(toPath)) {
        fs.mkdirSync(toPath, { recursive: true });
      }

      // Copy all files and directories
      const items = fs.readdirSync(fromPath);

      for (const item of items) {
        const fromItem = path.join(fromPath, item);
        const toItem = path.join(toPath, item);
        const stat = fs.statSync(fromItem);

        if (stat.isDirectory()) {
          this.moveDirectory(fromItem, toItem);
        } else {
          fs.copyFileSync(fromItem, toItem);
          this.movedFiles.push(toItem);
          this.info(`Moved file: ${fromItem} -> ${toItem}`);
        }
      }

      return true;
    } catch (error) {
      this.error(`Failed to move directory ${fromPath} to ${toPath}: ${error.message}`);
      return false;
    }
  }

  /**
   * Remove a directory recursively
   */
  removeDirectory(dirPath) {
    try {
      if (fs.existsSync(dirPath)) {
        fs.rmSync(dirPath, { recursive: true, force: true });
        this.info(`Removed directory: ${dirPath}`);
      }
    } catch (error) {
      this.error(`Failed to remove directory ${dirPath}: ${error.message}`);
    }
  }

  /**
   * Main function to move a Go module
   */
  async moveModule(fromPath, toPath) {
    this.info(`Starting module move: ${fromPath} -> ${toPath}`);

    const moduleName = this.getModuleName();
    if (moduleName) {
      this.info(`Detected module name: ${moduleName}`);
    }

    // Normalize paths
    const normalizedFrom = this.normalizePath(fromPath);
    const normalizedTo = this.normalizePath(toPath);

    // Check if source exists
    if (!this.isGoModulePath(normalizedFrom) && !fs.existsSync(normalizedFrom)) {
      this.error(`Source path does not exist: ${normalizedFrom}`);
      return false;
    }

    // Handle file system path movement
    if (!this.isGoModulePath(normalizedFrom)) {
      this.info(`Moving files from ${normalizedFrom} to ${normalizedTo}`);

      // Move the directory
      const moveSuccess = this.moveDirectory(normalizedFrom, normalizedTo);
      if (!moveSuccess) {
        return false;
      }

      // Remove the source directory after successful move
      this.removeDirectory(normalizedFrom);
    }

    // Update imports in all Go files
    this.info('Updating import statements...');
    const goFiles = this.findGoFiles();
    let updatedFiles = 0;

    for (const file of goFiles) {
      if (this.updateImportsInFile(file, fromPath, toPath, moduleName)) {
        updatedFiles++;
      }
    }

    this.info(`Module move completed successfully!`);
    this.info(`- Files moved: ${this.movedFiles.length}`);
    this.info(`- Import statements updated: ${updatedFiles}`);

    if (this.errors.length > 0) {
      this.warn(`Encountered ${this.errors.length} errors during move:`);
      this.errors.forEach(error => console.log(`  - ${error}`));
    }

    return true;
  }
}

// Main execution
async function main() {
  const args = process.argv.slice(2);

  if (args.length !== 2) {
    console.error('Usage: node move-go-module.js <from> <to>');
    console.error('');
    console.error('Examples:');
    console.error('  node move-go-module.js internal/common pkg/common');
    console.error('  node move-go-module.js ./internal/api ./pkg/api');
    console.error('  node move-go-module.js github.com/old/repo/internal/utils github.com/new/repo/pkg/utils');
    process.exit(1);
  }

  const [fromPath, toPath] = args;
  const mover = new GoModuleMover();

  try {
    const success = await mover.moveModule(fromPath, toPath);
    process.exit(success ? 0 : 1);
  } catch (error) {
    console.error('Unexpected error:', error.message);
    process.exit(1);
  }
}

if (require.main === module) {
  main();
}

module.exports = GoModuleMover;