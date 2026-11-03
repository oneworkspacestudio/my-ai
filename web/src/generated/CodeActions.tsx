import { Button, Row, Spacer, Text } from "@/components/ui"

export function View({ data, onAction }: { data: any; onAction: (name: string) => void }) {
  return (
    <Row variant="code-actions">
      <Text variant="code-language">{data.language}</Text>
      <Spacer />
      <Button onClick={() => onAction("wrap")} label={data.wrapLabel} icon="WrapText" variant="compact" pressed={data.wrapped} />
      <Button onClick={() => onAction("download")} label="Download code" icon="Download" variant="compact" />
      <Button onClick={() => onAction("copy")} label={data.copyLabel} icon="Copy" variant="compact" />
      {data.error && (
        <Text variant="status">{data.error}</Text>
      )}
    </Row>
  )
}
