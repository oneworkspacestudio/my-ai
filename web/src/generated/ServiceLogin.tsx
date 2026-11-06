import { Alert, Button, Card, Heading, Image, Input, Page, Text } from "@/components/ui"

export function ServiceLogin({ data, onAction }: { data: any; onAction: (name: string) => void }) {
  return (
    <Page variant="login">
      <Card variant="login">
        <Image src="./ai-service.svg" width={64} height={64} alt="AI Service" />
        <Text variant="eyebrow">ONE WORKSPACE</Text>
        <Heading level={1}>Your shared local AI</Heading>
        <Text>Manage the active model and share a private request queue across users and apps.</Text>
        {data.oidc ? (
          <Button onClick={() => onAction("oidc")} variant="primary">Continue with MyAccounts</Button>
        ) : (
          <>
            <Input name="email" type="email" placeholder="Email address" />
            <Input name="password" type="password" placeholder="Password" />
            <Button onClick={() => onAction("login")} variant="primary">Sign in</Button>
          </>
        )}
        {data.error && (
          <Alert>{data.error}</Alert>
        )}
      </Card>
    </Page>
  )
}
