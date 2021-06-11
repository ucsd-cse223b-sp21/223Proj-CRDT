import matplotlib.pyplot as plt

x_axis = []
y_axis = []
with open("Llatency.csv") as fi:
    for line in fi:
            a,b,c = line.split(',')
            x_axis.append(int(b))
            y_axis.append(float(c))
x_axis_1 = []
y_axis_1 = []
with open("Nlatency.csv") as fi:
    for line in fi:
            a,b,c = line.split(',')
            x_axis_1.append(int(b))
            y_axis_1.append(float(c))
#print(x_axis[:10], y_axis[:10])
plt.plot(x_axis,y_axis,label="Local Latency")
plt.plot(x_axis_1,y_axis_1, label="Network Latency")
plt.legend()
plt.show()