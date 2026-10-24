import { Alert, Button, Card, Divider, Heading, Icon, Image, Input, Link, Page, Row, Select, Spacer, Stack, Text } from "@/components/ui"

export function Workspace({ data, onAction }: { data: any; onAction: (name: string) => void }) {
  return (
    <Page variant={data.theme}>
      <Stack variant="rail">
        <Image src="./logo.svg" width={36} height={36} alt="My AI" />
        {data.rail.map((item, i) => (
          <Button key={i} onClick={() => onAction("navigate")} item={item.action} icon={item.icon} label={item.label} variant={item.variant} />
        ))}
        <Spacer />
        <Button onClick={() => onAction("theme")} icon="SunMoon" label="Change theme" variant="rail" />
        <Button onClick={() => onAction("account")} icon="CircleUserRound" label="Your account" variant="rail" />
      </Stack>
      <Stack variant="sidebar">
        <Row variant="brand">
          <Button onClick={() => onAction("surface")} icon="ChevronDown" variant="surface">{data.surfaceLabel}</Button>
          <Spacer />
          <Button onClick={() => onAction("search")} icon="Search" label="Search sessions" variant="icon" />
          <Button onClick={() => onAction("new")} icon="SquarePen" label="New conversation" variant="icon" />
        </Row>
        {data.surfaceMenu && (
          <Card variant="surface-menu">
            <Button onClick={() => onAction("chat")} icon="MessageSquare" label="Chat" variant="menu" />
            <Text>Explore ideas, ask anything</Text>
            <Button onClick={() => onAction("code")} icon="CodeXml" label="Code" variant="menu" />
            <Text>Build with projects and Codex</Text>
          </Card>
        )}
        <Button onClick={() => onAction("new")} icon="Plus" label="New conversation" variant="new" />
        <Input name="search" placeholder="Search conversations" variant="search" />
        <Row variant="section-title">
          <Text>PROJECTS</Text>
          <Spacer />
          <Button onClick={() => onAction("projects")} icon="FolderPlus" label="Manage projects" variant="icon" />
        </Row>
        {data.groups.map((item, i) => (
          <Stack key={i} variant="project-group">
            <Button onClick={() => onAction("select-project")} item={item.id} icon="Folder" label={item.name} variant="project" />
            {item.sessions.map((item, i) => (
              <Button key={i} onClick={() => onAction("open")} item={item.id} label={item.label} variant={item.variant} />
            ))}
          </Stack>
        ))}
        <Text variant="section-title">{data.sessionGroup}</Text>
        {data.sessions.map((item, i) => (
          <Button key={i} onClick={() => onAction("open")} item={item.id} label={item.label} variant={item.variant} />
        ))}
        <Spacer />
        <Row variant="sidebar-footer">
          <Text variant="dot">●</Text>
          <Text>{data.storageLabel}</Text>
          <Spacer />
          <Button onClick={() => onAction("archives")} icon="Archive" label="Show archived" variant="icon" />
        </Row>
      </Stack>
      <Stack variant="main">
        <Row variant="topbar">
          <Button onClick={() => onAction("mobile-menu")} icon="PanelLeft" label="Toggle sidebar" variant="icon" />
          <Icon name={data.modeIcon} size={18} />
          <Text variant="title">{data.title}</Text>
          <Spacer />
          <Button onClick={() => onAction("diff")} icon="GitCompareArrows" label="Changes" disabled={data.noProject} variant="subtle" />
          <Button onClick={() => onAction("session-menu")} icon="Ellipsis" label="Conversation actions" variant="icon" />
          <Button onClick={() => onAction("apps")} icon="Grid2X2" label="Apps" variant="icon" />
        </Row>
        <Stack variant="conversation">
          {data.empty && (
            <Stack variant="welcome">
              <Card variant="welcome-icon"><Icon name="Sparkles" size={34} /></Card>
              <Text variant="eyebrow">YOUR SPACE TO THINK & BUILD</Text>
              <Heading level={1}>{data.welcomeTitle}</Heading>
              <Text>{data.welcomeDescription}</Text>
              <Row variant="suggestions">
                <Button onClick={() => onAction("suggestion-chat")} icon="Lightbulb" label="Explore an idea" variant="suggestion" />
                <Button onClick={() => onAction("suggestion-code")} icon="CodeXml" label="Build something" variant="suggestion" />
                <Button onClick={() => onAction("models")} icon="Cpu" label="Use a local model" variant="suggestion" />
              </Row>
            </Stack>
          )}
          {data.messages.map((item, i) => (
            <Card key={i} variant={item.variant}>
              <Row variant="message-heading">
                <Icon name={item.icon} size={18} />
                <Text variant="author">{item.author}</Text>
                <Spacer />
                <Text variant="muted">{item.model}</Text>
              </Row>
              <Text variant="markdown">{item.content}</Text>
              <Text variant="muted">{item.status}</Text>
            </Card>
          ))}
          {data.streaming && (
            <Card variant="assistant">
              <Row variant="message-heading">
                <Icon name="Sparkles" size={18} />
                <Text variant="author">My AI</Text>
                <Text variant="muted">{data.workingLabel}</Text>
              </Row>
              <Text variant="markdown">{data.partial}</Text>
            </Card>
          )}
          {data.hasEvents && (
            <Card variant="activity">
              <Heading level={3}>Activity</Heading>
              {data.events.map((item, i) => (
                <Card key={i} variant="event">
                  <Text variant="event-title">{item.method}</Text>
                  <Text variant="pre">{item.content}</Text>
                </Card>
              ))}
            </Card>
          )}
          {data.approvals.map((item, i) => (
            <Card key={i} variant="approval">
              <Row>
                <Icon name="ShieldQuestionMark" size={22} />
                <Heading level={3}>Approval required</Heading>
              </Row>
              <Text variant="pre">{item.content}</Text>
              <Row>
                <Button onClick={() => onAction("approve")} item={item.id} variant="primary">Approve once</Button>
                <Button onClick={() => onAction("deny")} item={item.id}>Deny</Button>
              </Row>
            </Card>
          ))}
        </Stack>
        <Stack variant="composer-wrap">
          {data.message && (
            <Alert>{data.message}</Alert>
          )}
          {data.hasAttachments && (
            <Row variant="attachments">
              {data.attachments.map((item, i) => (
                <Button key={i} onClick={() => onAction("remove-attachment")} item={item.id} icon="FileText" label={item.name} variant="subtle" />
              ))}
            </Row>
          )}
          <Card variant="composer">
            <Input name="prompt" type="textarea" placeholder={data.placeholder} variant="prompt" />
            <Row variant="composer-controls">
              <Button onClick={() => onAction("attach")} icon="Plus" label="Attach text or code" variant="icon" disabled={data.busy} />
              <Button onClick={() => onAction("policy")} icon="ShieldCheck" variant="permission">{data.policyLabel}</Button>
              <Spacer />
              <Select name="provider" options={data.providerOptions} disabled={data.busy} />
              <Select name="model" options={data.modelOptions} disabled={data.busy} />
              {data.codeMode && (
                <Button onClick={() => onAction("effort")} icon="Gauge" variant="effort">{data.effortLabel}</Button>
              )}
              {data.busy ? (
                <Button onClick={() => onAction("stop")} icon="Square" label="Stop" variant="send" />
              ) : (
                <Button onClick={() => onAction("send")} icon="ArrowUp" label="Send message" variant="send" disabled={data.cannotSend} />
              )}
            </Row>
            {data.effortOpen && (
              <Card variant="effort-popover">
                <Row>
                  <Icon name="Zap" size={18} />
                  <Text variant="author">Reasoning effort</Text>
                </Row>
                {data.efforts.map((item, i) => (
                  <Button key={i} onClick={() => onAction("set-effort")} item={item.value} label={item.label} variant={item.variant} />
                ))}
              </Card>
            )}
          </Card>
          <Row variant="composer-caption">
            <Text>{data.providerNotice}</Text>
            <Spacer />
            <Text>Ctrl + Enter to send</Text>
          </Row>
        </Stack>
      </Stack>
      {data.modal && (
        <Card variant="modal">
          <Row variant="modal-heading">
            <Heading level={2}>{data.modalTitle}</Heading>
            <Spacer />
            <Button onClick={() => onAction("close")} icon="X" label="Close dialog" variant="icon" />
          </Row>
          {data.settingsView && (
            <>
              <Text>API keys are protected by Windows on this host. They are never saved in conversations or returned by the server.</Text>
              <Text variant="field-label">OpenAI API key</Text>
              <Input name="openaiKey" type="password" placeholder="Enter OpenAI API key" />
              <Button onClick={() => onAction("save-openai")} variant="primary">Save OpenAI connection</Button>
              <Text variant="field-label">DeepSeek API key</Text>
              <Input name="deepseekKey" type="password" placeholder="Enter DeepSeek API key" />
              <Button onClick={() => onAction("save-deepseek")} variant="primary">Save DeepSeek connection</Button>
              <Divider />
              <Text>{data.storageDetail}</Text>
              <Text>Custom endpoints, cloud budgets and voice/image attachments are unavailable in this preview.</Text>
            </>
          )}
          {data.modelsView && (
            <>
              <Row>
                <Text>Models run on the machine hosting My AI.</Text>
                <Spacer />
                <Button onClick={() => onAction("refresh")} icon="RefreshCw" label="Refresh models" variant="subtle" />
              </Row>
              {data.providerStatuses.map((item, i) => (
                <Text key={i} variant="status">{item.label}</Text>
              ))}
              {data.models.map((item, i) => (
                <Card key={i} variant="model-card">
                  <Row>
                    <Icon name="Cpu" size={22} />
                    <Stack>
                      <Text variant="author">{item.name}</Text>
                      <Text variant="muted">{item.detail}</Text>
                    </Stack>
                    <Spacer />
                    <Button onClick={() => onAction("use-model")} item={item.key}>Use</Button>
                    {item.local && (
                      <Button onClick={() => onAction("delete-model")} item={item.id} icon="Trash2" label="Remove model" variant="icon" />
                    )}
                  </Row>
                </Card>
              ))}
              <Divider />
              <Heading level={3}>Download an Ollama model</Heading>
              <Text>Check a model’s licence and hardware needs in the official library. Ollama chooses storage using its existing service configuration. My AI does not change or move that library.</Text>
              <Link href="https://ollama.com/library">Browse the official model library ↗</Link>
              <Link href="https://ollama.com/download/windows">Get Ollama for Windows ↗</Link>
              <Input name="pullModel" placeholder="Exact model tag, for example qwen3:0.6b" />
              <Button onClick={() => onAction("pull")} variant="primary" disabled={data.pullBusy}>Download model</Button>
              {data.pullBusy && (
                <Button onClick={() => onAction("cancel-pull")}>Cancel download</Button>
              )}
              <Text variant="pre">{data.pullProgress}</Text>
            </>
          )}
          {data.projectsView && (
            <>
              <Text>Approve a dedicated host folder. Code reads project files; workspace-write can change files after you select it. Remote members cannot register folders or run Code mode.</Text>
              <Input name="projectName" placeholder="Project name" />
              <Input name="projectPath" placeholder="Absolute folder on the host PC" />
              <Button onClick={() => onAction("add-project")} icon="FolderPlus" variant="primary">Approve project folder</Button>
              {data.projects.map((item, i) => (
                <Row key={i} variant="project-card">
                  <Stack>
                    <Text variant="author">{item.name}</Text>
                    <Text variant="muted">{item.path}</Text>
                  </Stack>
                  <Spacer />
                  <Button onClick={() => onAction("choose-project")} item={item.id}>Open</Button>
                  <Button onClick={() => onAction("remove-project")} item={item.id} icon="X" label="Remove project" variant="icon" />
                </Row>
              ))}
            </>
          )}
          {data.sessionView && (
            <>
              <Input name="title" placeholder="Conversation title" />
              <Button onClick={() => onAction("rename")}>Rename</Button>
              <Button onClick={() => onAction("pin")}>{data.pinLabel}</Button>
              <Button onClick={() => onAction("archive")}>{data.archiveLabel}</Button>
              <Button onClick={() => onAction("fork")}>Fork conversation</Button>
              <Button onClick={() => onAction("export-json")}>Export JSON</Button>
              <Button onClick={() => onAction("export-markdown")}>Export Markdown</Button>
              <Button onClick={() => onAction("import")}>Import JSON conversation</Button>
              <Button onClick={() => onAction("retry")} disabled={data.busy}>Retry last prompt</Button>
              <Button onClick={() => onAction("delete")} variant="danger">Move to private trash</Button>
            </>
          )}
          {data.diffView && (
            <>
              <Text>Changes come directly from your approved Git project. Staging and revert are unavailable in this preview.</Text>
              <Text variant="pre">{data.gitStatus}</Text>
              <Text variant="diff">{data.gitDiff}</Text>
            </>
          )}
          {data.accountView && (
            <>
              <Text variant="author">{data.accountName}</Text>
              <Text>{data.accountDetail}</Text>
              <Text>My AI is independent. Codex uses app-owned configuration and explicitly entered API credentials.</Text>
              <Button onClick={() => onAction("logout")}>Sign out</Button>
            </>
          )}
          {data.message && (
            <Alert>{data.message}</Alert>
          )}
        </Card>
      )}
    </Page>
  )
}
