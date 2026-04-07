import { z } from "zod";

export const commandEnvelopeSchema = z.discriminatedUnion("type", [
  z.object({
    commandId: z.string(),
    type: z.literal("command.startThread"),
    payload: z.object({
      machineId: z.string(),
      title: z.string(),
    }),
  }),
  z.object({
    commandId: z.string(),
    type: z.literal("command.startTurn"),
    payload: z.object({
      threadId: z.string(),
      prompt: z.string(),
    }),
  }),
  z.object({
    commandId: z.string(),
    type: z.literal("command.toggleSkill"),
    payload: z.object({
      machineId: z.string(),
      resourceId: z.string(),
      enabled: z.boolean(),
    }),
  }),
]);
