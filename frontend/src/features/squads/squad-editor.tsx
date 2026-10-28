import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'

/** Edits descriptive data only; these notes are not executable instructions. */
export function SquadEditor({
  initial,
  busy,
  onSave,
  onCancel,
}: {
  initial?: { name: string; description: string; instructions: string }
  busy: boolean
  onSave: (value: { name: string; description: string; instructions: string }) => void
  onCancel: () => void
}) {
  const { t } = useTranslation()
  const [name, setName] = useState(initial?.name ?? '')
  const [description, setDescription] = useState(initial?.description ?? '')
  const [instructions, setInstructions] = useState(initial?.instructions ?? '')
  return (
    <form
      className="space-y-3 rounded-lg border p-4"
      onSubmit={(event) => {
        event.preventDefault()
        onSave({ name, description, instructions })
      }}
    >
      <label className="block space-y-1">
        {t('squad.name')}
        <Input required maxLength={200} value={name} onChange={(e) => setName(e.target.value)} />
      </label>
      <label className="block space-y-1">
        {t('squad.description')}
        <Textarea
          maxLength={2000}
          value={description}
          onChange={(e) => setDescription(e.target.value)}
        />
      </label>
      <label className="block space-y-1">
        {t('squad.instructions')}
        <Textarea
          maxLength={8000}
          value={instructions}
          onChange={(e) => setInstructions(e.target.value)}
        />
      </label>
      <div className="flex gap-2">
        <Button type="submit" disabled={busy || !name.trim()}>
          {t('squad.save')}
        </Button>
        <Button type="button" variant="outline" onClick={onCancel}>
          {t('squad.cancel')}
        </Button>
      </div>
    </form>
  )
}
