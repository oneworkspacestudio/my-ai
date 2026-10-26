import { Alert, Button, Card, Divider, Heading, Icon, Image, Input, Link, Metric, Page, Row, Select, Spacer, Stack, Text } from "@/components/ui"

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
            <Text>Build with your projects</Text>
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
        {data.hasActiveDownloads && (
          <Button onClick={() => onAction("models")} icon="Download" variant="download-banner">{data.downloadSummary}</Button>
        )}
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
          {data.hasSources && (
            <Card variant="sources">
              <Heading level={3}>Sources</Heading>
              {data.sources.map((item, i) => (
                <Link key={i} href={item.url}>{item.title}</Link>
              ))}
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
            {data.isCodeMode && (
              <Row variant="composer-project">
                <Button onClick={() => onAction("projects")} icon="FolderOpen" variant="project-picker" disabled={data.busy} label={data.projectLabel} />
                <Button onClick={() => onAction("create-project")} icon="FolderPlus" variant="subtle" disabled={data.busy}>New project</Button>
              </Row>
            )}
            <Input name="prompt" type="textarea" placeholder={data.placeholder} variant="prompt" />
            <Row variant="composer-controls">
              <Button onClick={() => onAction("tools")} icon="Wrench" label="Tools & skills" variant="icon" disabled={data.busy} />
              <Button onClick={() => onAction("attach")} icon="Plus" label="Attach text or code" variant="icon" disabled={data.busy} />
              <Button onClick={() => onAction("policy")} icon="ShieldCheck" variant="permission" disabled={data.busy}>{data.policyLabel}</Button>
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
              <Input name="modelSearch" placeholder="Filter model names" />
              {data.modelGroups.map((item, i) => (
                <Card key={i} variant="provider-group">
                  <Button onClick={() => onAction("toggle-provider")} item={item.id} icon={item.icon} expanded={item.expanded} label={item.label} variant="provider-toggle" />
                  {item.expanded && (
                    <>
                      <Text variant="status">{item.status}</Text>
                      <Stack variant="model-list">
                        {item.models.map((item, i) => (
                          <Card key={i} variant="model-card">
                            <Row>
                              <Icon name="Cpu" size={22} />
                              <Stack>
                                <Text variant="author">{item.name}</Text>
                                <Text variant="muted">{item.detail}</Text>
                              </Stack>
                              <Spacer />
                              <Button onClick={() => onAction("use-model")} item={item.key}>Use</Button>
                              {item.canRemove && (
                                <Button onClick={() => onAction("delete-model")} item={item.id} icon="Trash2" label="Remove model" variant="icon" />
                              )}
                            </Row>
                          </Card>
                        ))}
                      </Stack>
                    </>
                  )}
                </Card>
              ))}
              <Divider />
              <Heading level={3}>Download an Ollama model</Heading>
              <Text>Check a model’s licence and hardware needs in the official library. Ollama chooses storage using its existing service configuration. My AI does not change or move that library.</Text>
              <Link href="https://ollama.com/library">Browse the official model library ↗</Link>
              <Link href="https://ollama.com/download/windows">Get Ollama for Windows ↗</Link>
              <Text>Downloads continue on this server when you close the page. Progress and installed local models are shared by all users of this server.</Text>
              {data.isOwner && (
                <>
                  <Input name="pullModel" placeholder="Exact model tag, for example qwen3:0.6b" />
                  <Button onClick={() => onAction("pull")} variant="primary">Download model</Button>
                </>
              )}
              <Heading level={3}>Server downloads</Heading>
              {data.downloads.map((item, i) => (
                <Card key={i} variant="download-row">
                  <Text variant="author">{item.label}</Text>
                  <Metric value={item.completed} max={item.total} label={item.label} />
                  <Text variant="muted">{item.bytes}</Text>
                  <Text variant="status">{item.detail}</Text>
                  {item.canCancel && (
                    <Button onClick={() => onAction("cancel-pull")} item={item.id}>Cancel download</Button>
                  )}
                  {item.canRetry && (
                    <Button onClick={() => onAction("retry-pull")} item={item.model}>Retry download</Button>
                  )}
                </Card>
              ))}
            </>
          )}
          {data.toolsView && (
            <>
              <Heading level={3}>Web search</Heading>
              <Text>{data.searchProvider}</Text>
              <Text>Turn on Search web in the composer to search before answering. It works in Chat and projects with every chat model. Only your question is searched; attached files are kept private.</Text>
              <Link href={data.searchSource}>SearXNG source and licences ↗</Link>
              <Text>{data.toolNotice}</Text>
              <Text>Chat has live time, calculator, public web search/page reading, and private conversation/attachment reads. You can read your own projects. Automatic tool use depends on the selected model; explicit commands work with any chat model.</Text>
              {data.toolCommands.map((item, i) => (
                <Button key={i} onClick={() => onAction("tool-command")} item={item.value} label={item.value} variant="menu" />
              ))}
              {data.toolList.map((item, i) => (
                <Card key={i} variant="tool-card">
                  <Text variant="author">{item.name}</Text>
                  <Text variant="muted">{item.description}</Text>
                </Card>
              ))}
              <Text>Ask normally. My AI searches the web when current information is needed and shows its sources. You can ask it to search, or say “don’t search the web” to keep the conversation offline. Attachment contents are never added to automatic search queries.</Text>
              <Text>Codex uses the same private SearXNG search through My AI’s included tools. Project editing stays inside the selected project and saves to your private Git repository.</Text>
            </>
          )}
          {data.projectsView && (
            <>
              <Text>Choose a project for this conversation, or use General to get started.</Text>
              <Button onClick={() => onAction("general-project")} icon="Folder" variant="primary" disabled={data.projectBusy}>Use General project</Button>
              <Text>Give your project a name. We'll create its folder in My Drive and a private repository in Git. My AI saves completed changes to both automatically.</Text>
              <Input name="projectName" placeholder="Project name" />
              <Button onClick={() => onAction("add-project")} icon="FolderPlus" variant="primary" disabled={data.projectBusy}>{data.createProjectLabel}</Button>
              {data.projects.map((item, i) => (
                <Card key={i} variant="project-card">
                  <Row>
                    <Icon name="FolderCode" size={22} />
                    <Text variant="author">{item.name}</Text>
                  </Row>
                  <Text variant="muted">{item.location}</Text>
                  <Text variant="status">{item.status}</Text>
                  {item.hasLinks && (
                    <Row variant="project-links">
                      <Link href={item.driveUrl}>Open in My Drive</Link>
                      <Link href={item.url}>View in Git</Link>
                    </Row>
                  )}
                  <Row variant="project-actions">
                    <Button onClick={() => onAction("choose-project")} item={item.id} disabled={item.disabledOpen} variant="primary">Open project</Button>
                    {item.managed && (
                      <Button onClick={() => onAction("save-project")} item={item.id} disabled={data.projectBusy}>{item.saveLabel}</Button>
                    )}
                  </Row>
                </Card>
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
              <Text>Review the files changed in your project. Completed AI changes are saved to Git automatically.</Text>
              <Text variant="pre">{data.gitStatus}</Text>
              <Text variant="diff">{data.gitDiff}</Text>
            </>
          )}
          {data.accountView && (
            <>
              <Text variant="author">{data.accountName}</Text>
              <Text>{data.accountDetail}</Text>
              <Text>Your projects and conversations stay connected to your account.</Text>
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
