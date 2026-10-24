import { Alert, Button, Card, Heading, Image, Input, Page, Text } from "@/components/ui"

export function Login({ data, onAction }: { data: any; onAction: (name: string) => void }) {
  return (
    <Page variant="login">
      <Card variant="login">
        <Image src="./logo.svg" width={64} height={64} alt="My AI" />
        <Text variant="eyebrow">ONE WORKSPACE</Text>
        <Heading level={1}>A little curiosity. Endless possibility.</Heading>
        <Text>Chat, think and build with your own AI workspace.</Text>
        {data.oidc ? (
          <Button onClick={() => onAction("oidc")} variant="primary">Continue with MyAccounts</Button>
        ) : (
          <>
            <Input name="email" type="email" placeholder="Email address" />
            <Input name="password" type="password" placeholder="Password" />
            <Button onClick={() => onAction("login")} variant="primary">{data.loginLabel}</Button>
          </>
        )}
        {data.error && (
          <Alert>{data.error}</Alert>
        )}
      </Card>
    </Page>
  )
}
