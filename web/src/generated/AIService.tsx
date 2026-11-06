import { Alert, Button, Card, Grid, Heading, Image, Input, Link, Page, Row, Select, Spacer, Stack, Text } from "@/components/ui"

export function AIService({ data, onAction }: { data: any; onAction: (name: string) => void }) {
  return (
    <Page variant="service">
      <Row>
        <Image src="./ai-service.svg" width={48} height={48} alt="AI Service" />
        <Stack>
          <Text variant="eyebrow">ONE WORKSPACE</Text>
          <Heading level={1}>AI Service</Heading>
        </Stack>
        <Spacer />
        <Link href={data.workspaceURL} variant="apps">Apps</Link>
        <Link href={data.myaiURL}>My AI</Link>
        <Button onClick={() => onAction("logout")}>Sign out</Button>
      </Row>
      <Text>One local model. A shared queue for your people and apps.</Text>
      {data.message && (
        <Alert>{data.message}</Alert>
      )}
      <Grid>
        <Card>
          <Text variant="eyebrow">MODEL</Text>
          <Heading level={2}>{data.phase}</Heading>
          <Text>{data.modelLabel}</Text>
          <Text>{data.dependency}</Text>
        </Card>
        <Card>
          <Text variant="eyebrow">REQUESTS</Text>
          <Heading level={2}>{data.queueLabel}</Heading>
          <Text>{data.activeLabel}</Text>
          <Text>{data.finishedLabel}</Text>
        </Card>
      </Grid>
      {data.owner && (
        <Card>
          <Heading level={2}>Keep a model ready</Heading>
          <Text>Choose a model already installed in Ollama. Enabling keeps it in memory until you turn the service off.</Text>
          <Text>Shared inference uses this model for every user and app. Turn off to cancel active and waiting requests and release the model.</Text>
          <Text>Active model</Text>
          <Select name="model" label="Active model" options={data.modelOptions} disabled={data.busy} />
          <Row variant="service-actions">
            <Button onClick={() => onAction("enable")} variant="primary" disabled={data.busy}>Save and turn on</Button>
            <Button onClick={() => onAction("disable")} disabled={data.busy}>Turn off and unload</Button>
            <Button onClick={() => onAction("refresh")} disabled={data.busy}>Refresh models</Button>
          </Row>
        </Card>
      )}
      <Card>
        <Heading level={2}>Shared request queue</Heading>
        <Text>{data.queuePolicy}</Text>
        <Text>Up to 64 waiting requests and 8 per app/user. Replies stream as they arrive. Prompts and replies are never saved by AI Service.</Text>
        <Text>In My AI, choose the active local model. My AI sends local chat and coding requests through this queue automatically. Cloud providers keep their existing settings.</Text>
      </Card>
      {data.owner && (
        <Card>
          <Heading level={2}>Connected apps</Heading>
          <Text>{data.managedLabel}</Text>
          <Text>Other server apps can receive a separate credential. Copy it once into that app's private settings. Never put it in a browser or a public file.</Text>
          <Input name="clientID" label="App ID" placeholder="App ID, for example my-workflow" />
          <Button onClick={() => onAction("createClient")} disabled={data.busy}>Create app credential</Button>
          {data.newKey && (
            <>
              <Text variant="pre">{data.newKey}</Text>
              <Button onClick={() => onAction("copyKey")}>Copy credential</Button>
              <Button onClick={() => onAction("dismissKey")}>Hide credential</Button>
            </>
          )}
          {data.clients.map((item, i) => (
            <Row key={i}>
              <Text>{item.id}</Text>
              <Spacer />
              <Button onClick={() => onAction("revoke")} item={item} disabled={data.busy}>Revoke</Button>
            </Row>
          ))}
          <Text>Connect server apps using the API described in the AI Service setup guide. Credentials can be revoked here at any time.</Text>
        </Card>
      )}
    </Page>
  )
}
