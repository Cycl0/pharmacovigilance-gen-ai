import networkx as nx
import matplotlib.pyplot as plt
import pymongo
from pymongo import MongoClient
from dotenv import load_dotenv
import scipy
import os
import json

# .env
load_dotenv()
MONGO_URI = os.getenv('MONGODB_URI')

# Connect to MongoDB
client = MongoClient(MONGO_URI)
db = client['bluesky_data']
collection = db['medications']

# Create an empty graph
G = nx.Graph()

for doc in collection.find():
    medication = doc['name']
    if medication == "X":
      continue

    if not G.has_node(medication):
        G.add_node(medication, type='medication')
        for i, adrs in doc.get('adrs', []):
            adr = adrs[i]
            if not G.has_node(adr):
                G.add_node(adr, type='adr')
                G.add_edge(adr, medication, relationship='related')

# Visualize the graph with different colors for different node types
# plt.figure(figsize=(200, 200))

# Use spring layout with k parameter to increase node spacing
pos = nx.spring_layout(G, k=0.1, seed=42, iterations=200, scale=2.0)

# Calculate node degrees for size mapping
degrees = dict(G.degree())
max_degree = max(degrees.values())

# Custom scaling function for node sizes
def scale_size(degree, max_degree):
    return 50 + 300 * (degree / max_degree)**0.5  # Non-linear scaling

# Create node sizes and labels based on degree
node_sizes = []
node_colors = []
filtered_labels = {}
min_degree_for_label = 10  # Only label nodes with at least this many connections

for node in G.nodes():
    node_type = G.nodes[node].get('type')
    if node_type == 'user':
        node_colors.append('blue')
    elif node_type == 'medication':
        node_colors.append('green')
    else:  # ADR
        node_colors.append('red')

    degree = degrees[node]
    node_sizes.append(scale_size(degree, max_degree))

    if degree >= min_degree_for_label:
        filtered_labels[node] = node

min_degree_to_show = 3  # Remove nodes with fewer than 3 connections
G_filtered = G.subgraph([n for n in G.nodes() if G.degree(n) >= min_degree_to_show])

# Draw elements with different parameters
nx.draw_networkx_nodes(
    G, pos,
    node_size=node_sizes,
    node_color=node_colors,
    alpha=0.7,
    edgecolors='black',
    linewidths=0.3
)

nx.draw_networkx_edges(
    G, pos,
    width=0.3,  # Thinner edges
    alpha=0.2,  # More transparent
    edge_color='gray'
)

# Only draw labels for important nodes
nx.draw_networkx_labels(
    G, pos,
    labels=filtered_labels,
    font_size=8,
    font_color='black',
    alpha=0.8
)

# Improve legend
legend_elements = [
    plt.Line2D([0], [0], marker='o', color='w', label='Medications', markerfacecolor='green', markersize=8),
    plt.Line2D([0], [0], marker='o', color='w', label='ADRs', markerfacecolor='red', markersize=8)
]

# plt.legend(handles=legend_elements, loc='best', fontsize=10)

# # Add degree-based title
# plt.title(f"Medication-ADR-User Network (Label threshold: ≥{min_degree_for_label} connections)", fontsize=12)
# plt.axis('off')

# # Use tight layout
# plt.tight_layout()
# plt.savefig("/output/graph.png", dpi=300, bbox_inches='tight')

# Calculate degrees
degrees = dict(G.degree())
nx.set_node_attributes(G, degrees, "degree")

# Convert positions to string format
for node in G.nodes():
    if node in pos:  # Check if position exists
        x, y = pos[node]
        # Format coordinates with high precision
        G.nodes[node]["position"] = f"{float(x):.6f} {float(y):.6f}"
    else:
        # Provide default position if missing
        G.nodes[node]["position"] = "0.000000 0.000000"

# Ensure all attributes are properly formatted for GEXF
for node, data in G.nodes(data=True):
    # Convert degree to integer to ensure compatibility
    if "degree" in data:
        G.nodes[node]["degree"] = int(data["degree"])

# Write GEXF with specific attribute configurations
nx.write_gexf(
    G,
    "/output/medicamentos_adrs_only_network.gexf",
    encoding="utf-8",
    prettyprint=True,
    version="1.2draft"
)
