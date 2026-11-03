import { Button, Row } from "@/components/ui"

export function View({ data, onAction }: { data: any; onAction: (name: string) => void }) {
  return (
    <Row variant="jump-latest"><Button onClick={() => onAction("latest")} icon="ArrowDown" label="Jump to latest" variant="compact" /></Row>
  )
}
