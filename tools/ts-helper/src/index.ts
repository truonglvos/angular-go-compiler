import * as fs from "fs";
import * as path from "path";
import * as ts from "typescript";

// ComponentMetadata extracted from a component file
interface ComponentMetadata {
  className: string;
  selector: string;
  template?: string;
  templateUrl?: string;
  inputs: string[];
  outputs: string[];
  filePath: string;
}

interface HelperResult {
  components: ComponentMetadata[];
  errors: string[];
}

/**
 * Extract component metadata from a TypeScript source file
 */
function extractComponentMetadata(filePath: string): ComponentMetadata[] {
  const sourceText = fs.readFileSync(filePath, "utf-8");
  const sourceFile = ts.createSourceFile(
    filePath,
    sourceText,
    ts.ScriptTarget.Latest,
    true
  );

  const components: ComponentMetadata[] = [];

  function visit(node: ts.Node): void {
    // Look for class decorators
    if (ts.isClassDeclaration(node)) {
      const decorators = ts.getDecorators(node);
      if (decorators && decorators.length > 0) {
        for (const decorator of decorators) {
          // Check if decorator is @Component
          if (isComponentDecorator(decorator)) {
            const metadata = extractMetadataFromDecorator(
              decorator,
              node.name?.text || "Unknown"
            );
            if (metadata) {
              metadata.filePath = filePath;
              components.push(metadata);
            }
          }
        }
      }
    }

    ts.forEachChild(node, visit);
  }

  visit(sourceFile);
  return components;
}

function isComponentDecorator(decorator: ts.Decorator): boolean {
  if (!ts.isCallExpression(decorator.expression)) {
    return false;
  }

  const expr = decorator.expression.expression;
  if (ts.isIdentifier(expr)) {
    return expr.text === "Component";
  }

  return false;
}

function extractMetadataFromDecorator(
  decorator: ts.Decorator,
  className: string
): ComponentMetadata | null {
  if (!ts.isCallExpression(decorator.expression)) {
    return null;
  }

  const call = decorator.expression;
  if (call.arguments.length === 0) {
    return null;
  }

  const arg = call.arguments[0];
  if (!ts.isObjectLiteralExpression(arg)) {
    return null;
  }

  const metadata: ComponentMetadata = {
    className,
    selector: "",
    inputs: [],
    outputs: [],
    filePath: "",
  };

  for (const prop of arg.properties) {
    if (!ts.isPropertyAssignment(prop)) continue;

    const propName = prop.name?.getText?.() || "";
    const propValue = prop.initializer;

    if (propName === "selector" && ts.isStringLiteral(propValue)) {
      metadata.selector = propValue.text;
    } else if (propName === "template" && ts.isStringLiteral(propValue)) {
      metadata.template = propValue.text;
    } else if (propName === "templateUrl" && ts.isStringLiteral(propValue)) {
      metadata.templateUrl = propValue.text;
    } else if (propName === "inputs" && ts.isArrayLiteralExpression(propValue)) {
      metadata.inputs = extractStringArrayElements(propValue);
    } else if (propName === "outputs" && ts.isArrayLiteralExpression(propValue)) {
      metadata.outputs = extractStringArrayElements(propValue);
    }
  }

  return metadata;
}

function extractStringArrayElements(arr: ts.ArrayLiteralExpression): string[] {
  const result: string[] = [];
  for (const element of arr.elements) {
    if (ts.isStringLiteral(element)) {
      result.push(element.text);
    }
  }
  return result;
}

/**
 * Scan a directory recursively for .ts files and extract component metadata
 */
function scanProject(projectPath: string): HelperResult {
  const components: ComponentMetadata[] = [];
  const errors: string[] = [];

  function walkDir(dirPath: string): void {
    try {
      const files = fs.readdirSync(dirPath);
      for (const file of files) {
        const fullPath = path.join(dirPath, file);
        const stat = fs.statSync(fullPath);

        if (stat.isDirectory()) {
          // Skip node_modules, dist, etc.
          if (!file.startsWith(".") && file !== "node_modules" && file !== "dist") {
            walkDir(fullPath);
          }
        } else if (file.endsWith(".ts") && !file.endsWith(".spec.ts")) {
          try {
            const found = extractComponentMetadata(fullPath);
            components.push(...found);
          } catch (err) {
            errors.push(`Error in ${fullPath}: ${err}`);
          }
        }
      }
    } catch (err) {
      errors.push(`Error walking ${dirPath}: ${err}`);
    }
  }

  walkDir(projectPath);
  return { components, errors };
}

// Main entry point - read project path from argv and output JSON to stdout
function main(): void {
  const projectPath = process.argv[2] || ".";

  try {
    const result = scanProject(projectPath);
    console.log(JSON.stringify(result, null, 2));
    process.exit(0);
  } catch (err) {
    console.error(JSON.stringify({ components: [], errors: [String(err)] }));
    process.exit(1);
  }
}

main();
