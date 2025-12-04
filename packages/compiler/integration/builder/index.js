#!/usr/bin/env node

const { createBuilder } = require("@angular-devkit/architect");
const { spawn } = require("child_process");
const path = require("path");
const fs = require("fs");
const { Observable } = require("rxjs");

// Find the Go binary or use 'go run' as fallback
// Tìm từ project root để hoạt động tốt khi builder được cài vào node_modules
function findGoBinary(projectRoot) {
  // Try multiple possible locations relative to project root
  const possibleBinaryPaths = [
    // Built binary in workspace root (nếu project nằm trong workspace)
    path.resolve(projectRoot, "../../bin/ngc-go"),
    // Binary trong project root
    path.resolve(projectRoot, "bin/ngc-go"),
    // Binary trong cmd directory từ workspace root
    path.resolve(projectRoot, "../../cmd/ngc-go/ngc-go"),
    // Binary trong cmd directory từ project root
    path.resolve(projectRoot, "cmd/ngc-go/ngc-go"),
    // Binary trong packages/compiler/cmd/ngc-go (old location)
    path.resolve(projectRoot, "../../packages/compiler/cmd/ngc-go/ngc-go"),
  ];

  for (const binaryPath of possibleBinaryPaths) {
    if (fs.existsSync(binaryPath)) {
      return { type: "binary", path: binaryPath };
    }
  }
  
  // Fallback to 'go run' - tìm package directory từ project root
  const possibleGoPackageDirs = [
    path.resolve(projectRoot, "../../cmd/ngc-go"),
    path.resolve(projectRoot, "cmd/ngc-go"),
    path.resolve(projectRoot, "../../packages/compiler/cmd/ngc-go"),
  ];

  for (const goPackageDir of possibleGoPackageDirs) {
    const mainGoPath = path.join(goPackageDir, "main.go");
    if (fs.existsSync(mainGoPath)) {
      return { type: "go-run", path: goPackageDir };
    }
  }
  
  return null;
}

/**
 * Angular Builder implementation for ngc-go compiler
 */
function build(options, context) {
  return new Observable((observer) => {
    const projectRoot = context.workspaceRoot;
    const tsConfig = options.tsConfig || "tsconfig.json";
    const tsConfigPath = path.resolve(projectRoot, tsConfig);

    context.logger.info(`🔨 Building Angular project with ngc-go...`);
    context.logger.info(`   Project root: ${projectRoot}`);
    context.logger.info(`   TypeScript config: ${tsConfigPath}`);

    const goBinary = findGoBinary(projectRoot);
    if (!goBinary) {
      observer.error(
        new Error(
          "Could not find ngc-go binary or main.go file. " +
          "Please ensure ngc-go is built or main.go exists in cmd/ngc-go/"
        )
      );
      return;
    }

    // Determine output path
    const outputPath = options.outputPath || "dist/ngc-go";
    const resolvedOutputPath = path.isAbsolute(outputPath)
      ? outputPath
      : path.resolve(projectRoot, outputPath);
    
    context.logger.info(`   Output path: ${resolvedOutputPath}`);

    // Prepare arguments for ngc-go
    const args = ["compile", projectRoot, resolvedOutputPath];
    
    let child;
    if (goBinary.type === "binary") {
      context.logger.info(`   Using binary: ${goBinary.path}`);
      context.logger.info(`   Command: ${goBinary.path} ${args.join(' ')}`);
      child = spawn(goBinary.path, args, {
        cwd: projectRoot,
        stdio: ["inherit", "pipe", "pipe"],
      });
    } else {
      // When using 'go run', we need to run from the package directory
      // and include all .go files in that directory
      const packageDir = goBinary.path;
      const fullCommand = `go run . ${args.join(' ')}`;
      context.logger.info(`   Using: go run (from ${packageDir})`);
      context.logger.info(`   Command: ${fullCommand}`);
      child = spawn("go", ["run", ".", ...args], {
        cwd: packageDir,
        stdio: ["inherit", "pipe", "pipe"],
      });
    }

    // Forward stdout to logger
    child.stdout.on("data", (data) => {
      const output = data.toString();
      // Split by lines and log each line
      output.split('\n').forEach(line => {
        if (line.trim()) {
          context.logger.info(line);
        }
      });
    });

    // Forward stderr to logger
    child.stderr.on("data", (data) => {
      const output = data.toString();
      // Split by lines and log each line
      output.split('\n').forEach(line => {
        if (line.trim()) {
          context.logger.error(line);
        }
      });
    });

    child.on("close", (code) => {
      if (code === 0) {
        context.logger.info("✅ Build completed successfully");
        observer.next({ success: true });
        observer.complete();
      } else {
        context.logger.error(`❌ Build failed with exit code ${code}`);
        observer.error(new Error(`Build failed with exit code ${code}`));
      }
    });

    child.on("error", (err) => {
      context.logger.error(`Failed to start compiler: ${err.message}`);
      observer.error(err);
    });
  });
}

// Export the builder
module.exports = createBuilder(build);
