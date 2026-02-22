import os
import re
from pathlib import Path

# Config
MAX_FILE_SIZE = 5 * 1024 * 1024  # 5MB limit
EXCLUDE_PATTERNS = {
    'node_modules', 'venv', '__pycache__', 'dist', '.git', 
    '.DS_Store', 'structure.txt', 'structure.md', 'STRUCTURE.md',
    '.next', 'build', 'coverage', '.turbo', 'data'
}

def get_file_info(filepath: Path):
    """Extracts function/class names and a brief description from a file."""
    try:
        # Skip large files
        if filepath.stat().st_size > MAX_FILE_SIZE:
            return "File too large (>5MB)", []
        
        # Read with error handling for encoding issues
        content = filepath.read_text(encoding='utf-8', errors='ignore')
        
        # Extract description from first JSDoc or comment block
        description = "Source file."
        jsdoc_match = re.search(r'/\*\*[\s\S]*?\*/', content)
        if jsdoc_match:
            # Clean up JSDoc markers and extra whitespace
            desc_text = re.sub(r'/\*\*|\*/|\s*\*\s*', ' ', jsdoc_match.group())
            description = " ".join(desc_text.split())[:150]
        
        # Extract symbols with word boundaries to reduce false positives
        items = []
        
        # Function declarations: function name(
        items.extend(re.findall(r'\bfunction\s+([a-zA-Z_$][\w$]*)\s*\(', content))
        
        # Class declarations: class Name
        items.extend(re.findall(r'\bclass\s+([a-zA-Z_$][\w$]*)\s*[<{]', content))
        
        # Exports: export class/function/const name
        items.extend(re.findall(
            r'\bexport\s+(?:default\s+)?(?:class|function|const|let|var)\s+([a-zA-Z_$][\w$]*)', 
            content
        ))
        
        # Arrow functions: const name = (...) => (conservative pattern)
        # Only matches if assignment and arrow are on same line or close
        items.extend(re.findall(
            r'\b(?:const|let|var)\s+([a-zA-Z_$][\w$]*)\s*=\s*(?:async\s*)?\([^)]{0,100}\)\s*=>', 
            content
        ))
        
        # Method definitions (class methods, object methods)
        items.extend(re.findall(r'^\s+([a-zA-Z_$][\w$]*)\s*\([^)]*\)\s*\{', content, re.MULTILINE))
        
        unique_items = sorted(set(items))
        return description, unique_items
        
    except (IOError, PermissionError, OSError) as e:
        return f"Access denied ({e.__class__.__name__})", []
    except Exception as e:
        return f"Error: {str(e)[:50]}", []

def generate_md_structure(startpath: Path, exclude_patterns: set):
    """Generate markdown structure with correct indentation."""
    md_lines = [f"# 🏗️ {startpath.name} Structure\n"]
    
    for root, dirs, files in os.walk(startpath):
        # Filter directories in-place
        dirs[:] = [d for d in dirs if d not in exclude_patterns]
        
        # Calculate depth correctly
        rel_path = Path(root).relative_to(startpath)
        depth = len(rel_path.parts) - 1 if str(rel_path) != '.' else 0
        
        folder_name = Path(root).name if str(rel_path) != '.' else startpath.name
        
        # Folder indentation: 2 spaces per depth
        indent = "  " * depth
        md_lines.append(f"{indent}- 📁 **{folder_name}/**")
        
        # Files are indented one level deeper
        file_indent = "  " * (depth + 1)
        
        for f in sorted(files):
            if f in exclude_patterns or f.startswith('.'):
                continue
            
            filepath = Path(root) / f
            
            if f.endswith(('.ts', '.tsx', '.js', '.jsx', '.mjs', '.cjs')):
                desc, items = get_file_info(filepath)
                
                # File header with description
                md_lines.append(f"{file_indent}- 📄 **{f}**")
                md_lines.append(f"{file_indent}  > *{desc}*")
                
                # Symbols list (limited to 8)
                if items:
                    displayed = items[:8]
                    item_str = ", ".join([f"`{i}`" for i in displayed])
                    if len(items) > 8:
                        item_str += f", ... (+{len(items)-8} more)"
                    md_lines.append(f"{file_indent}  - ⚙️ {item_str}")
            else:
                md_lines.append(f"{file_indent}- 📄 {f}")
                
    return "\n".join(md_lines)

def main():
    # Use the directory containing the script as the root
    script_dir = Path(__file__).parent.absolute()
    repo_path = script_dir.resolve()
    output_file = (script_dir / 'STRUCTURE.md').resolve()
    
    if not repo_path.exists():
        print(f"❌ Error: {repo_path} not found")
        return
    
    if not repo_path.is_dir():
        print(f"❌ Error: {repo_path} is not a directory")
        return
    
    print(f"🔍 Scanning {repo_path}...")
    md_output = generate_md_structure(repo_path, EXCLUDE_PATTERNS)
    
    # Write output
    output_file.write_text(md_output, encoding='utf-8')
    print(f"✅ Successfully generated {output_file}")
    print(f"📊 Output size: {len(md_output)} characters")

if __name__ == "__main__":
    main()