export function SubmitButton({ loading, label }: { loading: boolean; label: string }) {
  return <button className="primary" disabled={loading}>{loading ? 'Processando...' : label}</button>
}
