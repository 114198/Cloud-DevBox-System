/**
 * CollaborativeEditor Component
 * Real-time collaborative code editor using Monaco Editor with multi-cursor support
 * Requirements: 4.1, 4.2 - Real-time collaboration with cursor synchronization
 */

import { useEffect, useRef, useState, useCallback } from 'react'
import { Spin, Alert, message } from 'antd'
import { LoadingOutlined } from '@ant-design/icons'
import { useCollaborationStore, useRemoteCursors, useCurrentUserColor } from '@/stores/collaboration'
import type { CursorPosition, Selection, EditOperation } from '@/services/collaboration'

// Monaco Editor types
interface IStandaloneCodeEditor {
  getValue(): string
  setValue(value: string): void
  getModel(): ITextModel | null
  getPosition(): IPosition | null
  getSelection(): ISelection | null
  setPosition(position: IPosition): void
  setSelection(selection: ISelection): void
  onDidChangeCursorPosition(listener: (e: { position: IPosition }) => void): IDisposable
  onDidChangeCursorSelection(listener: (e: { selection: ISelection }) => void): IDisposable
  onDidChangeModelContent(listener: (e: IModelContentChangedEvent) => void): IDisposable
  createDecorationsCollection(decorations: IModelDeltaDecoration[]): IEditorDecorationsCollection
  deltaDecorations(oldDecorations: string[], newDecorations: IModelDeltaDecoration[]): string[]
  focus(): void
  layout(): void
}

interface ITextModel {
  getValue(): string
  getLineContent(lineNumber: number): string
  getLineCount(): number
  getOffsetAt(position: IPosition): number
  getPositionAt(offset: number): IPosition
}

interface IPosition {
  lineNumber: number
  column: number
}

interface ISelection {
  startLineNumber: number
  startColumn: number
  endLineNumber: number
  endColumn: number
}

interface IModelContentChangedEvent {
  changes: Array<{
    range: {
      startLineNumber: number
      startColumn: number
      endLineNumber: number
      endColumn: number
    }
    rangeOffset: number
    rangeLength: number
    text: string
  }>
  isFlush: boolean
}

interface IModelDeltaDecoration {
  range: {
    startLineNumber: number
    startColumn: number
    endLineNumber: number
    endColumn: number
  }
  options: {
    className?: string
    glyphMarginClassName?: string
    hoverMessage?: { value: string }[]
    beforeContentClassName?: string
    afterContentClassName?: string
    inlineClassName?: string
  }
}

interface IEditorDecorationsCollection {
  set(decorations: IModelDeltaDecoration[]): void
  clear(): void
}

interface IDisposable {
  dispose(): void
}

// Monaco loader
declare global {
  interface Window {
    monaco?: {
      editor: {
        create(element: HTMLElement, options: Record<string, unknown>): IStandaloneCodeEditor
        defineTheme(name: string, theme: Record<string, unknown>): void
      }
    }
  }
}

interface CollaborativeEditorProps {
  environmentId: string
  filePath: string
  initialContent?: string
  language?: string
  readOnly?: boolean
  onContentChange?: (content: string) => void
  onSave?: (content: string) => void
  height?: string | number
}

// Remote cursor decoration styles
const createCursorStyles = () => {
  const styleId = 'collaborative-editor-styles'
  if (document.getElementById(styleId)) return

  const style = document.createElement('style')
  style.id = styleId
  style.textContent = `
    .remote-cursor {
      position: relative;
      border-left: 2px solid;
      margin-left: -1px;
    }
    .remote-cursor::before {
      content: attr(data-username);
      position: absolute;
      top: -18px;
      left: -2px;
      padding: 2px 6px;
      font-size: 10px;
      font-weight: 500;
      color: white;
      border-radius: 3px 3px 3px 0;
      white-space: nowrap;
      z-index: 100;
    }
    .remote-selection {
      opacity: 0.3;
    }
    ${Array.from({ length: 12 }, (_, i) => {
      const colors = [
        '#FF6B6B', '#4ECDC4', '#45B7D1', '#96CEB4',
        '#FFEAA7', '#DDA0DD', '#98D8C8', '#F7DC6F',
        '#BB8FCE', '#85C1E9', '#F8B500', '#00CED1',
      ]
      return `
        .remote-cursor-${i} { border-color: ${colors[i]}; }
        .remote-cursor-${i}::before { background-color: ${colors[i]}; }
        .remote-selection-${i} { background-color: ${colors[i]}; }
      `
    }).join('\n')}
  `
  document.head.appendChild(style)
}

export default function CollaborativeEditor({
  environmentId,
  filePath,
  initialContent = '',
  language = 'typescript',
  readOnly = false,
  onContentChange,
  onSave,
  height = '500px',
}: CollaborativeEditorProps) {
  const editorRef = useRef<HTMLDivElement>(null)
  const monacoEditorRef = useRef<IStandaloneCodeEditor | null>(null)
  const decorationsRef = useRef<string[]>([])
  const isRemoteChangeRef = useRef(false)
  const lastContentRef = useRef(initialContent)

  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const {
    isConnected,
    isConnecting,
    connectionError,
    connect,
    disconnect,
    sendCursorUpdate,
    sendSelectionUpdate,
    sendEdit,
    pendingOperations,
  } = useCollaborationStore()

  const remoteCursors = useRemoteCursors()
  const currentUserColor = useCurrentUserColor()

  // Load Monaco Editor
  useEffect(() => {
    createCursorStyles()
    loadMonaco()

    return () => {
      if (monacoEditorRef.current) {
        // Monaco doesn't have a dispose method in our simplified interface
        monacoEditorRef.current = null
      }
    }
  }, [])

  const loadMonaco = async () => {
    try {
      // Check if Monaco is already loaded
      if (window.monaco) {
        initializeEditor()
        return
      }

      // Load Monaco from CDN
      const script = document.createElement('script')
      script.src = 'https://cdn.jsdelivr.net/npm/monaco-editor@0.45.0/min/vs/loader.js'
      script.async = true
      
      script.onload = () => {
        // @ts-expect-error - require is defined by Monaco loader
        window.require.config({
          paths: { vs: 'https://cdn.jsdelivr.net/npm/monaco-editor@0.45.0/min/vs' }
        })
        // @ts-expect-error - require is defined by Monaco loader
        window.require(['vs/editor/editor.main'], () => {
          initializeEditor()
        })
      }

      script.onerror = () => {
        setError('Failed to load Monaco Editor')
        setIsLoading(false)
      }

      document.head.appendChild(script)
    } catch (err) {
      setError('Failed to initialize editor')
      setIsLoading(false)
    }
  }

  const initializeEditor = () => {
    if (!editorRef.current || !window.monaco) return

    // Define custom theme
    window.monaco.editor.defineTheme('devbox-dark', {
      base: 'vs-dark',
      inherit: true,
      rules: [],
      colors: {
        'editor.background': '#1e1e2e',
        'editor.foreground': '#cdd6f4',
        'editorCursor.foreground': currentUserColor || '#f5e0dc',
        'editor.lineHighlightBackground': '#313244',
        'editorLineNumber.foreground': '#6c7086',
        'editor.selectionBackground': '#45475a',
        'editor.inactiveSelectionBackground': '#313244',
      },
    })

    const editor = window.monaco.editor.create(editorRef.current, {
      value: initialContent,
      language,
      theme: 'devbox-dark',
      readOnly,
      automaticLayout: true,
      minimap: { enabled: true },
      fontSize: 14,
      lineNumbers: 'on',
      wordWrap: 'on',
      scrollBeyondLastLine: false,
      renderWhitespace: 'selection',
      tabSize: 2,
      insertSpaces: true,
      cursorBlinking: 'smooth',
      cursorSmoothCaretAnimation: 'on',
      smoothScrolling: true,
      padding: { top: 10, bottom: 10 },
    })

    monacoEditorRef.current = editor

    // Set up event listeners
    setupEditorListeners(editor)

    setIsLoading(false)

    // Connect to collaboration session
    connectToSession()
  }

  const setupEditorListeners = (editor: IStandaloneCodeEditor) => {
    // Cursor position change
    editor.onDidChangeCursorPosition((e) => {
      if (isRemoteChangeRef.current) return

      const position: CursorPosition = {
        filePath,
        line: e.position.lineNumber,
        column: e.position.column,
        offset: editor.getModel()?.getOffsetAt(e.position),
      }
      sendCursorUpdate(position)
    })

    // Selection change
    editor.onDidChangeCursorSelection((e) => {
      if (isRemoteChangeRef.current) return

      const selection: Selection = {
        filePath,
        startLine: e.selection.startLineNumber,
        startColumn: e.selection.startColumn,
        endLine: e.selection.endLineNumber,
        endColumn: e.selection.endColumn,
      }

      // Only send if there's an actual selection (not just cursor)
      if (
        selection.startLine !== selection.endLine ||
        selection.startColumn !== selection.endColumn
      ) {
        sendSelectionUpdate(selection)
      }
    })

    // Content change
    editor.onDidChangeModelContent((e) => {
      if (isRemoteChangeRef.current) return

      const content = editor.getValue()
      lastContentRef.current = content
      onContentChange?.(content)

      // Send edit operations
      e.changes.forEach((change) => {
        if (change.text) {
          // Insert operation
          sendEdit(filePath, 'insert', change.rangeOffset, change.text)
        }
        if (change.rangeLength > 0) {
          // Delete operation
          sendEdit(filePath, 'delete', change.rangeOffset, undefined, change.rangeLength)
        }
      })
    })

    // Keyboard shortcuts
    // @ts-expect-error - KeyMod and KeyCode are available on monaco
    editor.addCommand(window.monaco.KeyMod.CtrlCmd | window.monaco.KeyCode.KeyS, () => {
      onSave?.(editor.getValue())
      message.success('已保存')
    })
  }

  const connectToSession = async () => {
    try {
      await connect(environmentId)
    } catch (err) {
      console.error('Failed to connect to collaboration session:', err)
    }
  }

  // Apply remote edits
  useEffect(() => {
    if (!monacoEditorRef.current || pendingOperations.length === 0) return

    const editor = monacoEditorRef.current
    const model = editor.getModel()
    if (!model) return

    isRemoteChangeRef.current = true

    pendingOperations.forEach((operation: EditOperation) => {
      if (operation.filePath !== filePath) return

      const position = model.getPositionAt(operation.position)

      if (operation.type === 'insert' && operation.content) {
        // Apply insert
        const range = {
          startLineNumber: position.lineNumber,
          startColumn: position.column,
          endLineNumber: position.lineNumber,
          endColumn: position.column,
        }
        // @ts-expect-error - executeEdits is available on editor
        editor.executeEdits('remote', [{
          range,
          text: operation.content,
          forceMoveMarkers: true,
        }])
      } else if (operation.type === 'delete' && operation.length) {
        // Apply delete
        const endPosition = model.getPositionAt(operation.position + operation.length)
        const range = {
          startLineNumber: position.lineNumber,
          startColumn: position.column,
          endLineNumber: endPosition.lineNumber,
          endColumn: endPosition.column,
        }
        // @ts-expect-error - executeEdits is available on editor
        editor.executeEdits('remote', [{
          range,
          text: '',
          forceMoveMarkers: true,
        }])
      }
    })

    isRemoteChangeRef.current = false

    // Clear pending operations (this should be done in the store)
    useCollaborationStore.setState({ pendingOperations: [] })
  }, [pendingOperations, filePath])

  // Update remote cursor decorations
  const updateRemoteCursorDecorations = useCallback(() => {
    if (!monacoEditorRef.current) return

    const editor = monacoEditorRef.current
    const decorations: IModelDeltaDecoration[] = []
    const colorIndexMap = new Map<string, number>()
    let colorIndex = 0

    remoteCursors.forEach((cursor, userId) => {
      if (cursor.position.filePath !== filePath) return

      // Assign color index
      if (!colorIndexMap.has(userId)) {
        colorIndexMap.set(userId, colorIndex++)
      }
      const idx = colorIndexMap.get(userId) || 0

      // Cursor decoration
      decorations.push({
        range: {
          startLineNumber: cursor.position.line,
          startColumn: cursor.position.column,
          endLineNumber: cursor.position.line,
          endColumn: cursor.position.column + 1,
        },
        options: {
          className: `remote-cursor remote-cursor-${idx % 12}`,
          hoverMessage: [{ value: cursor.displayName || cursor.username }],
          beforeContentClassName: `remote-cursor-flag remote-cursor-${idx % 12}`,
        },
      })

      // Selection decoration
      if (cursor.selection && cursor.selection.filePath === filePath) {
        decorations.push({
          range: {
            startLineNumber: cursor.selection.startLine,
            startColumn: cursor.selection.startColumn,
            endLineNumber: cursor.selection.endLine,
            endColumn: cursor.selection.endColumn,
          },
          options: {
            className: `remote-selection remote-selection-${idx % 12}`,
          },
        })
      }
    })

    decorationsRef.current = editor.deltaDecorations(decorationsRef.current, decorations)
  }, [remoteCursors, filePath])

  useEffect(() => {
    updateRemoteCursorDecorations()
  }, [updateRemoteCursorDecorations])

  // Cleanup on unmount
  useEffect(() => {
    return () => {
      disconnect()
    }
  }, [disconnect])

  // Handle window resize
  useEffect(() => {
    const handleResize = () => {
      monacoEditorRef.current?.layout()
    }
    window.addEventListener('resize', handleResize)
    return () => window.removeEventListener('resize', handleResize)
  }, [])

  if (error) {
    return (
      <Alert
        type="error"
        message="编辑器加载失败"
        description={error}
        showIcon
      />
    )
  }

  return (
    <div className="collaborative-editor" style={{ height, position: 'relative' }}>
      {/* Loading overlay */}
      {(isLoading || isConnecting) && (
        <div
          style={{
            position: 'absolute',
            top: 0,
            left: 0,
            right: 0,
            bottom: 0,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            background: 'rgba(0, 0, 0, 0.5)',
            zIndex: 10,
          }}
        >
          <Spin
            indicator={<LoadingOutlined style={{ fontSize: 24 }} spin />}
            tip={isConnecting ? '正在连接协作会话...' : '加载编辑器...'}
          />
        </div>
      )}

      {/* Connection error */}
      {connectionError && (
        <Alert
          type="warning"
          message="协作连接失败"
          description={connectionError}
          showIcon
          closable
          style={{ marginBottom: 8 }}
        />
      )}

      {/* Connection status indicator */}
      {!isLoading && (
        <div
          style={{
            position: 'absolute',
            top: 8,
            right: 8,
            zIndex: 5,
            display: 'flex',
            alignItems: 'center',
            gap: 6,
            padding: '4px 8px',
            background: 'rgba(0, 0, 0, 0.6)',
            borderRadius: 4,
            fontSize: 12,
          }}
        >
          <span
            style={{
              width: 8,
              height: 8,
              borderRadius: '50%',
              background: isConnected ? '#52c41a' : '#ff4d4f',
            }}
          />
          <span style={{ color: '#fff' }}>
            {isConnected ? '已连接' : '未连接'}
          </span>
        </div>
      )}

      {/* Editor container */}
      <div
        ref={editorRef}
        style={{
          height: '100%',
          width: '100%',
          borderRadius: 8,
          overflow: 'hidden',
        }}
      />
    </div>
  )
}
