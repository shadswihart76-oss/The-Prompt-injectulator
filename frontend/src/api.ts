import type { Assessment, AssessmentRequest } from './types';

export async function generateAssessment(req: AssessmentRequest): Promise<Assessment> {
  const res = await fetch('/api/assessments', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(req),
  });

  const data = await res.json();

  if (!res.ok) {
    throw new Error(data.error ?? `Request failed with status ${res.status}`);
  }

  return data as Assessment;
}
