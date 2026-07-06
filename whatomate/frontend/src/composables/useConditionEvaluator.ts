/**
 * Safe expression evaluator for flow conditions
 * Supports: ==, !=, >, <, >=, <=, &&, ||, !, contains, startsWith, endsWith
 * NO eval() - uses tokenization and AST parsing
 */

type TokenType =
  | 'STRING'
  | 'NUMBER'
  | 'BOOLEAN'
  | 'IDENTIFIER'
  | 'OPERATOR'
  | 'LPAREN'
  | 'RPAREN'
  | 'EOF'

interface Token {
  type: TokenType
  value: string | number | boolean
}

interface ASTNode {
  type: string
  value?: any
  left?: ASTNode
  right?: ASTNode
  operator?: string
  operand?: ASTNode
  name?: string
  args?: ASTNode[]
}

class Tokenizer {
  private pos = 0
  private input: string

  constructor(input: string) {
    this.input = input.trim()
  }

  private isWhitespace(char: string): boolean {
    return /\s/.test(char)
  }

  private isDigit(char: string): boolean {
    return /[0-9]/.test(char)
  }

  private isAlpha(char: string): boolean {
    return /[a-zA-Z_]/.test(char)
  }

  private isAlphaNumeric(char: string): boolean {
    return /[a-zA-Z0-9_]/.test(char)
  }

  private peek(): string {
    return this.input[this.pos] || ''
  }

  private advance(): string {
    return this.input[this.pos++] || ''
  }

  private skipWhitespace(): void {
    while (this.isWhitespace(this.peek())) {
      this.advance()
    }
  }

  private readString(quote: string): string {
    let result = ''
    this.advance() // skip opening quote
    while (this.peek() && this.peek() !== quote) {
      if (this.peek() === '\\') {
        this.advance()
        const escaped = this.advance()
        if (escaped === 'n') result += '\n'
        else if (escaped === 't') result += '\t'
        else result += escaped
      } else {
        result += this.advance()
      }
    }
    this.advance() // skip closing quote
    return result
  }

  private readNumber(): number {
    let result = ''
    while (this.isDigit(this.peek()) || this.peek() === '.') {
      result += this.advance()
    }
    return parseFloat(result)
  }

  private readIdentifier(): string {
    let result = ''
    // Handle {{variable}} syntax
    if (this.peek() === '{' && this.input[this.pos + 1] === '{') {
      this.advance() // skip {
      this.advance() // skip {
      while (this.peek() && !(this.peek() === '}' && this.input[this.pos + 1] === '}')) {
        result += this.advance()
      }
      this.advance() // skip }
      this.advance() // skip }
      return result.trim()
    }

    while (this.isAlphaNumeric(this.peek())) {
      result += this.advance()
    }
    return result
  }

  tokenize(): Token[] {
    const tokens: Token[] = []

    while (this.pos < this.input.length) {
      this.skipWhitespace()

      if (this.pos >= this.input.length) break

      const char = this.peek()

      // String literals
      if (char === '"' || char === "'") {
        tokens.push({ type: 'STRING', value: this.readString(char) })
        continue
      }

      // Numbers
      if (this.isDigit(char)) {
        tokens.push({ type: 'NUMBER', value: this.readNumber() })
        continue
      }

      // Parentheses
      if (char === '(') {
        tokens.push({ type: 'LPAREN', value: '(' })
        this.advance()
        continue
      }

      if (char === ')') {
        tokens.push({ type: 'RPAREN', value: ')' })
        this.advance()
        continue
      }

      // Operators
      if (char === '!' && this.input[this.pos + 1] === '=') {
        tokens.push({ type: 'OPERATOR', value: '!=' })
        this.advance()
        this.advance()
        continue
      }

      if (char === '!' && this.input[this.pos + 1] !== '=') {
        tokens.push({ type: 'OPERATOR', value: '!' })
        this.advance()
        continue
      }

      if (char === '=' && this.input[this.pos + 1] === '=') {
        tokens.push({ type: 'OPERATOR', value: '==' })
        this.advance()
        this.advance()
        continue
      }

      if (char === '>' && this.input[this.pos + 1] === '=') {
        tokens.push({ type: 'OPERATOR', value: '>=' })
        this.advance()
        this.advance()
        continue
      }

      if (char === '<' && this.input[this.pos + 1] === '=') {
        tokens.push({ type: 'OPERATOR', value: '<=' })
        this.advance()
        this.advance()
        continue
      }

      if (char === '>') {
        tokens.push({ type: 'OPERATOR', value: '>' })
        this.advance()
        continue
      }

      if (char === '<') {
        tokens.push({ type: 'OPERATOR', value: '<' })
        this.advance()
        continue
      }

      if (char === '&' && this.input[this.pos + 1] === '&') {
        tokens.push({ type: 'OPERATOR', value: '&&' })
        this.advance()
        this.advance()
        continue
      }

      if (char === '|' && this.input[this.pos + 1] === '|') {
        tokens.push({ type: 'OPERATOR', value: '||' })
        this.advance()
        this.advance()
        continue
      }

      // Variable reference or keyword
      if (this.isAlpha(char) || char === '{') {
        const identifier = this.readIdentifier()

        // Keywords
        if (identifier === 'true') {
          tokens.push({ type: 'BOOLEAN', value: true })
        } else if (identifier === 'false') {
          tokens.push({ type: 'BOOLEAN', value: false })
        } else if (identifier === 'and') {
          tokens.push({ type: 'OPERATOR', value: '&&' })
        } else if (identifier === 'or') {
          tokens.push({ type: 'OPERATOR', value: '||' })
        } else if (identifier === 'not') {
          tokens.push({ type: 'OPERATOR', value: '!' })
        } else if (['contains', 'startsWith', 'endsWith'].includes(identifier)) {
          tokens.push({ type: 'OPERATOR', value: identifier })
        } else {
          tokens.push({ type: 'IDENTIFIER', value: identifier })
        }
        continue
      }

      // Unknown character, skip
      this.advance()
    }

    tokens.push({ type: 'EOF', value: '' })
    return tokens
  }
}

class Parser {
  private pos = 0
  private tokens: Token[]

  constructor(tokens: Token[]) {
    this.tokens = tokens
  }

  private current(): Token {
    return this.tokens[this.pos]
  }

  private advance(): Token {
    return this.tokens[this.pos++]
  }

  private expect(type: TokenType): Token {
    const token = this.current()
    if (token.type !== type) {
      throw new Error(`Expected ${type}, got ${token.type}`)
    }
    return this.advance()
  }

  parse(): ASTNode {
    return this.parseOr()
  }

  private parseOr(): ASTNode {
    let left = this.parseAnd()

    while (this.current().type === 'OPERATOR' && this.current().value === '||') {
      this.advance()
      const right = this.parseAnd()
      left = { type: 'BinaryOp', operator: '||', left, right }
    }

    return left
  }

  private parseAnd(): ASTNode {
    let left = this.parseNot()

    while (this.current().type === 'OPERATOR' && this.current().value === '&&') {
      this.advance()
      const right = this.parseNot()
      left = { type: 'BinaryOp', operator: '&&', left, right }
    }

    return left
  }

  private parseNot(): ASTNode {
    if (this.current().type === 'OPERATOR' && this.current().value === '!') {
      this.advance()
      const operand = this.parseNot()
      return { type: 'UnaryOp', operator: '!', operand }
    }

    return this.parseComparison()
  }

  private parseComparison(): ASTNode {
    let left = this.parseStringOp()

    const compOps = ['==', '!=', '>', '<', '>=', '<=']
    while (
      this.current().type === 'OPERATOR' &&
      compOps.includes(this.current().value as string)
    ) {
      const operator = this.advance().value as string
      const right = this.parseStringOp()
      left = { type: 'BinaryOp', operator, left, right }
    }

    return left
  }

  private parseStringOp(): ASTNode {
    let left = this.parsePrimary()

    const stringOps = ['contains', 'startsWith', 'endsWith']
    while (
      this.current().type === 'OPERATOR' &&
      stringOps.includes(this.current().value as string)
    ) {
      const operator = this.advance().value as string
      const right = this.parsePrimary()
      left = { type: 'StringOp', operator, left, right }
    }

    return left
  }

  private parsePrimary(): ASTNode {
    const token = this.current()

    if (token.type === 'NUMBER') {
      this.advance()
      return { type: 'Literal', value: token.value }
    }

    if (token.type === 'STRING') {
      this.advance()
      return { type: 'Literal', value: token.value }
    }

    if (token.type === 'BOOLEAN') {
      this.advance()
      return { type: 'Literal', value: token.value }
    }

    if (token.type === 'IDENTIFIER') {
      this.advance()
      return { type: 'Identifier', name: token.value as string }
    }

    if (token.type === 'LPAREN') {
      this.advance()
      const expr = this.parseOr()
      this.expect('RPAREN')
      return expr
    }

    throw new Error(`Unexpected token: ${token.type}`)
  }
}

class Evaluator {
  private variables: Record<string, any>

  constructor(variables: Record<string, any>) {
    this.variables = variables
  }

  evaluate(node: ASTNode): any {
    switch (node.type) {
      case 'Literal':
        return node.value

      case 'Identifier':
        return this.variables[node.name!] ?? ''

      case 'BinaryOp':
        return this.evaluateBinaryOp(node)

      case 'UnaryOp':
        return this.evaluateUnaryOp(node)

      case 'StringOp':
        return this.evaluateStringOp(node)

      default:
        throw new Error(`Unknown node type: ${node.type}`)
    }
  }

  private evaluateBinaryOp(node: ASTNode): any {
    const left = this.evaluate(node.left!)
    const right = this.evaluate(node.right!)

    switch (node.operator) {
      case '==':
        return left == right
      case '!=':
        return left != right
      case '>':
        return left > right
      case '<':
        return left < right
      case '>=':
        return left >= right
      case '<=':
        return left <= right
      case '&&':
        return Boolean(left) && Boolean(right)
      case '||':
        return Boolean(left) || Boolean(right)
      default:
        throw new Error(`Unknown operator: ${node.operator}`)
    }
  }

  private evaluateUnaryOp(node: ASTNode): any {
    const operand = this.evaluate(node.operand!)

    switch (node.operator) {
      case '!':
        return !operand
      default:
        throw new Error(`Unknown unary operator: ${node.operator}`)
    }
  }

  private evaluateStringOp(node: ASTNode): boolean {
    const left = String(this.evaluate(node.left!) ?? '')
    const right = String(this.evaluate(node.right!) ?? '')

    switch (node.operator) {
      case 'contains':
        return left.includes(right)
      case 'startsWith':
        return left.startsWith(right)
      case 'endsWith':
        return left.endsWith(right)
      default:
        throw new Error(`Unknown string operator: ${node.operator}`)
    }
  }
}

export function useConditionEvaluator() {
  /**
   * Evaluate a condition expression with given variables
   * @param expression - The condition expression (e.g., "name == 'John' && age > 18")
   * @param variables - Object containing variable values
   * @returns boolean result of the evaluation
   */
  function evaluateCondition(expression: string, variables: Record<string, any>): boolean {
    if (!expression || expression.trim() === '') {
      return false
    }

    try {
      const tokenizer = new Tokenizer(expression)
      const tokens = tokenizer.tokenize()
      const parser = new Parser(tokens)
      const ast = parser.parse()
      const evaluator = new Evaluator(variables)
      return Boolean(evaluator.evaluate(ast))
    } catch (error) {
      console.error('Condition evaluation error:', error)
      return false
    }
  }

  /**
   * Interpolate variables in a template string
   * @param template - Template string with {{variable}} placeholders
   * @param variables - Object containing variable values
   * @returns String with variables replaced
   */
  function interpolateVariables(template: string, variables: Record<string, any>): string {
    if (!template) return template

    return template.replace(/\{\{(\w+)\}\}/g, (match, varName) => {
      if (varName in variables) {
        return String(variables[varName] ?? '')
      }
      return match // Keep original if not found
    })
  }

  /**
   * Validate input against a regex pattern
   * @param input - User input to validate
   * @param pattern - Regex pattern string
   * @returns boolean indicating if input is valid
   */
  function validateInput(input: string, pattern: string): boolean {
    if (!pattern || pattern.trim() === '') {
      return true // No pattern means always valid
    }

    try {
      const regex = new RegExp(pattern)
      return regex.test(input)
    } catch (error) {
      console.error('Invalid regex pattern:', error)
      return true // Invalid pattern, allow input
    }
  }

  return {
    evaluateCondition,
    interpolateVariables,
    validateInput
  }
}
